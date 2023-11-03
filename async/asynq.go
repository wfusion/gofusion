package async

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/log"
	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/redis"

	rdsDrv "github.com/redis/go-redis/v9"

	pd "github.com/wfusion/gofusion/util/payload"
)

const (
	asyncqTaskTypenameField = "typename"
)

var (
	asynqLoggerType = reflect.TypeOf((*asynq.Logger)(nil)).Elem()
)

type asynqConsumer struct {
	*asynq.ServeMux

	appName string
	n       string
	c       *Conf

	mws      []asynq.MiddlewareFunc
	logger   asynq.Logger
	consumer *asynq.Server
}

func newAsynqConsumer(ctx context.Context, appName, name string, conf *Conf) Consumable {
	consumer := &asynqConsumer{appName: appName, n: name, c: conf}

	var rdsCli rdsDrv.UniversalClient
	switch conf.InstanceType {
	case instanceTypeRedis:
		rdsCli = redis.Use(ctx, conf.Instance, redis.AppName(appName))
	case instanceTypeDB:
		fallthrough
	default:
		panic(errors.Errorf("unknown instance type: %s", conf.InstanceType))
	}

	if consumer.logger == nil && utils.IsStrNotBlank(conf.Logger) {
		loggerType := inspect.TypeOf(conf.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			logger := log.Use(conf.LogInstance, log.AppName(appName))
			loggerValue.Interface().(customLogger).Init(logger, appName, name)
		}
		consumer.logger = loggerValue.Convert(asynqLoggerType).Interface().(asynq.Logger)
	}

	logLevel := asynq.LogLevel(0)
	utils.MustSuccess(logLevel.Set(conf.LogLevel))

	consumer.ServeMux = asynq.NewServeMux()
	asynqCfg := asynq.Config{
		Concurrency:    conf.ConsumerConcurrency,
		BaseContext:    context.Background,
		RetryDelayFunc: asynq.DefaultRetryDelayFunc,
		IsFailure:      nil,
		Queues:         nil,
		StrictPriority: conf.StrictPriority,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			taskName := "unknown"
			if task != nil {
				taskName = consumer.unformatTaskName(task.Type())
			}
			consumer.info(ctx, "handle task %s message error %s", taskName, err)
		}),
		Logger:                   consumer.logger,
		LogLevel:                 logLevel,
		ShutdownTimeout:          8 * time.Second,
		HealthCheckFunc:          func(err error) { consumer.warn(ctx, "health check check failed: %s", err) },
		HealthCheckInterval:      15 * time.Second,
		DelayedTaskCheckInterval: 5 * time.Second,
		GroupGracePeriod:         1 * time.Minute,
		GroupMaxDelay:            0,
		GroupMaxSize:             0,
		GroupAggregator:          nil,
		DisableRedisConnClose:    true,
	}
	if len(conf.Queues) > 0 {
		asynqCfg.Queues = make(map[string]int, len(conf.Queues))
		for _, queue := range conf.Queues {
			if _, ok := asynqCfg.Queues[queue.Name]; ok {
				panic(ErrDuplicatedQueueName)
			}
			if utils.IsStrBlank(queue.Name) {
				queue.Name = defaultQueue(appName)
			}
			asynqCfg.Queues[queue.Name] = queue.Level
		}
	} else {
		asynqCfg.Queues = map[string]int{defaultQueue(appName): 3}
	}

	consumer.consumer = asynq.NewServer(&asynqRedisConnOpt{UniversalClient: rdsCli}, asynqCfg)
	return consumer
}

func (a *asynqConsumer) Use(mws ...routerMiddleware) {
	for _, mw := range mws {
		a.mws = append(a.mws, a.adaptMiddleware(mw))
	}
}

func (a *asynqConsumer) Handle(pattern string, fn any, _ ...utils.OptionExtender) {
	if !a.c.Consumer {
		a.debug(context.Background(), "cannot handle task: consumer is not enabled")
		return
	}
	name := formatTaskName(a.appName, pattern)
	funcName := formatTaskName(a.appName, utils.GetFuncName(fn))

	callbackMapLock.Lock()
	defer callbackMapLock.Unlock()
	if callbackMap[a.appName] == nil {
		callbackMap[a.appName] = make(map[string]any)
	}
	if funcNameToTaskName[a.appName] == nil {
		funcNameToTaskName[a.appName] = make(map[string]string)
	}
	if _, ok := callbackMap[a.appName][name]; ok {
		panic(ErrDuplicatedHandlerName)
	}
	callbackMap[a.appName][name] = fn
	callbackMap[a.appName][funcName] = fn
	funcNameToTaskName[a.appName][funcName] = name

	typ, embed := wrapParams(fn)
	a.ServeMux.Handle(name, a.adaptAsynqHandlerFunc(fn, typ, embed))
	if name != funcName {
		a.ServeMux.Handle(funcName, a.adaptAsynqHandlerFunc(fn, typ, embed))
	}
}

func (a *asynqConsumer) HandleFunc(fn any, _ ...utils.OptionExtender) {
	if !a.c.Consumer {
		a.debug(context.Background(), "cannot handle task: consumer is not enabled")
		return
	}
	funcName := formatTaskName(a.appName, utils.GetFuncName(fn))

	callbackMapLock.Lock()
	defer callbackMapLock.Unlock()
	if callbackMap[a.appName] == nil {
		callbackMap[a.appName] = make(map[string]any)
	}
	if funcNameToTaskName[a.appName] == nil {
		funcNameToTaskName[a.appName] = make(map[string]string)
	}
	if _, ok := callbackMap[funcName]; ok {
		panic(ErrDuplicatedHandlerName)
	}
	callbackMap[a.appName][funcName] = fn

	typ, embed := wrapParams(fn)
	a.ServeMux.Handle(funcName, a.adaptAsynqHandlerFunc(fn, typ, embed))
}

func (a *asynqConsumer) Serve() (err error) {
	if !a.c.Consumer {
		return ErrConsumerDisabled
	}
	defer a.info(context.Background(), "consumer started")

	a.ServeMux.Use(a.gatewayMiddleware)
	a.ServeMux.Use(a.mws...)
	return a.consumer.Run(a.ServeMux)
}

func (a *asynqConsumer) Start() (err error) {
	if !a.c.Consumer {
		return ErrConsumerDisabled
	}

	defer a.info(context.Background(), "consumer started")

	a.ServeMux.Use(a.gatewayMiddleware)
	a.ServeMux.Use(a.mws...)

	return a.consumer.Start(a.ServeMux)
}

func (a *asynqConsumer) shutdown() (err error) {
	if a.consumer != nil {
		_, catchErr := utils.Catch(a.consumer.Shutdown)
		err = multierr.Append(err, errors.Cause(catchErr))
	}
	return
}

func (a *asynqConsumer) gatewayMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, raw *asynq.Task) (err error) {
		taskName := a.unformatTaskName(raw.Type())
		inspect.SetField(raw, asyncqTaskTypenameField, taskName)
		return next.ProcessTask(ctx, raw)
	})
}

func (a *asynqConsumer) adaptMiddleware(mw routerMiddleware) asynq.MiddlewareFunc {
	return func(asynqNext asynq.Handler) asynq.Handler {
		next := mw(a.adaptRouterHandlerFunc(asynqNext))
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			return next(ctx, a.newTask(t))
		})
	}
}

func (a *asynqConsumer) adaptAsynqHandlerFunc(h any, typ reflect.Type, embed bool) asynq.HandlerFunc {
	fn := utils.WrapFunc1[error](h)
	return func(ctx context.Context, task *asynq.Task) (err error) {
		ctx, data, _, err := pd.Unseal(task.Payload(), pd.Type(typ))
		if err != nil {
			return
		}
		params := unwrapParams(typ, embed, data)
		return fn(append([]any{ctx}, params...)...)
	}
}

func (a *asynqConsumer) adaptRouterHandlerFunc(h asynq.Handler) routerMiddlewareFunc {
	return func(ctx context.Context, raw Task) (err error) {
		return h.ProcessTask(ctx, a.newAsynqTask(raw))
	}
}

func (a *asynqConsumer) unformatTaskName(taskName string) (result string) {
	return strings.TrimPrefix(taskName, fmt.Sprintf("%s:async:", config.Use(a.appName).AppName()))
}

func (a *asynqConsumer) newTask(raw *asynq.Task) (t Task) {
	return &task{
		id:         raw.Type(),
		name:       raw.Type(),
		payload:    raw.Payload(),
		rawMessage: raw,
	}
}

func (a *asynqConsumer) newAsynqTask(raw Task) (t *asynq.Task) {
	return raw.RawMessage().(*asynq.Task)
}

type asynqProducer struct {
	*asynq.Client

	appName string
	n       string
	c       *Conf

	compressType  compress.Algorithm
	serializeType serialize.Algorithm
}

func newAsynqProducer(ctx context.Context, appName, name string, conf *Conf) Producable {
	var rdsCli rdsDrv.UniversalClient
	switch conf.InstanceType {
	case instanceTypeRedis:
		rdsCli = redis.Use(ctx, conf.Instance, redis.AppName(appName))
	case instanceTypeDB:
		fallthrough
	default:
		panic(errors.Errorf("unknown instance type: %s", conf.InstanceType))
	}

	producer := &asynqProducer{
		appName:       appName,
		n:             name,
		c:             conf,
		Client:        asynq.NewClient(&asynqRedisConnOpt{UniversalClient: rdsCli}),
		compressType:  compress.ParseAlgorithm(conf.MessageCompressType),
		serializeType: serialize.ParseAlgorithm(conf.MessageSerializeType),
	}
	// default serialize type
	if !producer.serializeType.IsValid() {
		producer.serializeType = serialize.AlgorithmGob
	}
	return producer
}

func (a *asynqProducer) Go(fn any, opts ...utils.OptionExtender) (err error) {
	var data any
	opt := utils.ApplyOptions[produceOption](opts...)
	if len(opt.args) > 0 {
		argType, embed := wrapParams(fn)
		data = setParams(argType, embed, opt.args...)
	}

	// get task name by func name
	funcName := formatTaskName(a.appName, utils.GetFuncName(fn))
	callbackMapLock.RLock()
	if mappingName, ok := funcNameToTaskName[a.appName][funcName]; ok {
		funcName = mappingName
	}
	callbackMapLock.RUnlock()

	ctx := context.Background()
	task, err := a.newTask(ctx, funcName, data)
	if err != nil {
		return
	}

	_, err = a.Client.EnqueueContext(ctx, task, a.parseOption(opt)...)
	if err != nil {
		return
	}
	return
}

func (a *asynqProducer) Goc(ctx context.Context, fn any, opts ...utils.OptionExtender) (err error) {
	var data any
	opt := utils.ApplyOptions[produceOption](opts...)
	if len(opt.args) > 0 {
		argType, embed := wrapParams(fn)
		data = setParams(argType, embed, opt.args...)
	}

	// get task name by func name
	funcName := formatTaskName(a.appName, utils.GetFuncName(fn))
	callbackMapLock.RLock()
	if mappingName, ok := funcNameToTaskName[a.appName][funcName]; ok {
		funcName = mappingName
	}
	callbackMapLock.RUnlock()

	task, err := a.newTask(ctx, funcName, data)
	if err != nil {
		return
	}

	_, err = a.Client.EnqueueContext(ctx, task, a.parseOption(opt)...)
	if err != nil {
		return
	}
	return
}

func (a *asynqProducer) Send(ctx context.Context, taskName string, data any, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[produceOption](opts...)
	task, err := a.newTask(ctx, formatTaskName(a.appName, taskName), data)
	if err != nil {
		return
	}

	_, err = a.Client.EnqueueContext(ctx, task, a.parseOption(opt)...)
	if err != nil {
		return
	}
	return
}

func (a *asynqProducer) parseOption(src *produceOption) (dst []asynq.Option) {
	if utils.IsStrNotBlank(src.id) {
		dst = append(dst, asynq.TaskID(src.id))
	}
	if utils.IsStrNotBlank(src.queue) {
		dst = append(dst, asynq.Queue(src.queue))
	} else if len(a.c.Queues) == 1 {
		dst = append(dst, asynq.Queue(a.c.Queues[0].Name))
	} else {
		dst = append(dst, asynq.Queue(defaultQueue(a.appName)))
	}
	if src.maxRetry > 0 {
		dst = append(dst, asynq.MaxRetry(src.maxRetry))
	}
	if !src.deadline.IsZero() {
		dst = append(dst, asynq.Deadline(src.deadline))
	}
	if src.timeout > 0 {
		dst = append(dst, asynq.Timeout(src.timeout))
	}
	if src.delayDuration > 0 {
		dst = append(dst, asynq.ProcessIn(src.timeout))
	}
	if !src.delayTime.IsZero() {
		dst = append(dst, asynq.ProcessAt(src.delayTime))
	}
	if src.retentionDuration > 0 {
		dst = append(dst, asynq.Retention(src.retentionDuration))
	}

	return
}

func (a *asynqProducer) newTask(ctx context.Context, taskName string, data any) (task *asynq.Task, err error) {
	payload, err := pd.Seal(data, pd.Context(ctx), pd.Serialize(a.serializeType), pd.Compress(a.compressType))
	if err != nil {
		return
	}

	task = asynq.NewTask(taskName, payload)
	return
}

type asynqRedisConnOpt struct{ rdsDrv.UniversalClient }

func (a *asynqRedisConnOpt) MakeRedisClient() any { return a.UniversalClient }

func formatTaskName(appName, taskName string) (result string) {
	return fmt.Sprintf("%s:async:%s", config.Use(appName).AppName(), taskName)
}

func defaultQueue(appName string) (result string) {
	return fmt.Sprintf("%s:async", config.Use(appName).AppName())
}
