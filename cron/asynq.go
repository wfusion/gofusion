package cron

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/redis"
	"github.com/wfusion/gofusion/routine"

	rdsDrv "github.com/redis/go-redis/v9"

	fmkCtx "github.com/wfusion/gofusion/context"
)

const (
	asyncqTaskPayloadField  = "payload"
	asyncqTaskTypenameField = "typename"
)

var (
	asynqLoggerType                     = reflect.TypeOf((*asynq.Logger)(nil)).Elem()
	asynqPeriodicTaskConfigProviderType = reflect.TypeOf((*asynq.PeriodicTaskConfigProvider)(nil)).Elem()
)

type asynqRouter struct {
	*asynq.ServeMux

	appName string

	l sync.RWMutex
	n string
	c *Conf

	mws     []asynq.MiddlewareFunc
	logger  asynq.Logger
	locker  lock.Lockable
	server  *asynq.Server
	trigger *asynq.PeriodicTaskManager

	id                    string
	lockDurations         map[string]time.Duration
	shouldShutdownServer  bool
	shouldShutdownTrigger bool
}

func newAsynq(ctx context.Context, appName, name string, conf *Conf) IRouter {
	r := &asynqRouter{
		appName:               appName,
		n:                     name,
		c:                     conf,
		lockDurations:         make(map[string]time.Duration, len(conf.Tasks)),
		shouldShutdownServer:  true,
		shouldShutdownTrigger: true,
	}
	if utils.IsStrBlank(r.c.Queue) {
		r.c.Queue = r.defaultQueue()
	}

	var rdsCli rdsDrv.UniversalClient
	switch conf.InstanceType {
	case instanceTypeRedis:
		rdsCli = redis.Use(ctx, conf.Instance, redis.AppName(appName))
	case instanceTypeMysql:
		fallthrough
	default:
		panic(errors.Errorf("unknown instance type: %s", conf.InstanceType))
	}

	if r.logger == nil && utils.IsStrNotBlank(conf.Logger) {
		loggerType := inspect.TypeOf(conf.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			logger := log.Use(conf.LogInstance, log.AppName(appName))
			loggerValue.Interface().(customLogger).Init(logger, appName, name)
		}
		r.logger = loggerValue.Convert(asynqLoggerType).Interface().(asynq.Logger)
	}
	if r.locker == nil && utils.IsStrNotBlank(conf.LockInstance) {
		r.locker = lock.Use(conf.LockInstance, lock.AppName(appName))
		if r.locker == nil {
			panic(errors.Errorf("locker instance not found: %s", conf.LockInstance))
		}
	}

	var provider asynq.PeriodicTaskConfigProvider
	if utils.IsStrNotBlank(conf.TaskLoader) {
		loaderType := inspect.TypeOf(conf.TaskLoader)
		if loaderType == nil {
			panic(errors.Errorf("%s not found", conf.TaskLoader))
		}
		provider = reflect.New(loaderType).
			Convert(asynqPeriodicTaskConfigProviderType).Interface().(asynq.PeriodicTaskConfigProvider)
	}

	logLevel := asynq.LogLevel(0)
	utils.MustSuccess(logLevel.Set(conf.LogLevel))

	wrapper := &asynqWrapper{r: r, n: r.n, appName: appName, cli: rdsCli, provider: provider}
	if conf.Trigger {
		r.initTrigger(ctx, wrapper, logLevel)
	}
	if conf.Server {
		r.initServer(ctx, wrapper, logLevel)
	}

	return r
}

func (a *asynqRouter) Use(mws ...routerMiddleware) {
	for _, mw := range mws {
		a.mws = append(a.mws, a.adaptMiddleware(mw))
	}
}

func (a *asynqRouter) Handle(pattern string, fn any, _ ...utils.OptionExtender) {
	if !a.c.Server {
		a.debug(context.Background(), "cannot handle task %s: client is not enabled", a.n)
		return
	}

	a.ServeMux.Handle(a.formatTaskName(pattern), a.adaptAsynqHandlerFunc(fn))
}

func (a *asynqRouter) Serve() (err error) {
	defer a.info(context.Background(), "scheduler is running")

	if a.c.Server {
		a.ServeMux.Use(a.gatewayMiddleware)
		a.ServeMux.Use(a.mws...)
	}

	if a.c.Trigger && !a.c.Server {
		return a.trigger.Run()
	}
	if !a.c.Trigger && a.c.Server {
		return a.server.Run(a.ServeMux)
	}

	a.shouldShutdownServer = false
	if err = a.trigger.Start(); err != nil {
		return
	}

	return a.server.Run(a.ServeMux)
}

func (a *asynqRouter) Start() (err error) {
	defer a.info(context.Background(), "scheduler started")

	if a.c.Trigger {
		if err = a.trigger.Start(); err != nil {
			return
		}
	}

	if a.c.Server {
		a.ServeMux.Use(a.gatewayMiddleware)
		a.ServeMux.Use(a.mws...)
		if err = a.server.Start(a.ServeMux); err != nil {
			return
		}
	}

	return
}

func (a *asynqRouter) shutdown() (err error) {
	if a.c.Trigger {
		_, catchErr := utils.Catch(a.trigger.Shutdown)
		err = multierr.Append(err, errors.Cause(catchErr))
	}
	if a.c.Server {
		_, catchErr := utils.Catch(a.server.Shutdown)
		err = multierr.Append(err, errors.Cause(catchErr))
	}
	return
}

func (a *asynqRouter) initTrigger(ctx context.Context, wrapper *asynqWrapper, logLevel asynq.LogLevel) {
	a.trigger = utils.Must(
		asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
			PeriodicTaskConfigProvider: wrapper,
			RedisConnOpt:               wrapper,
			SchedulerOpts: &asynq.SchedulerOpts{
				Logger:                a.logger,
				LogLevel:              logLevel,
				Location:              utils.Must(time.LoadLocation(a.c.Timezone)),
				DisableRedisConnClose: true,
				PreEnqueueFunc:        a.preEnqueueFunc(ctx),
				PostEnqueueFunc:       a.postEnqueueFunc(ctx),
				EnqueueErrorHandler: func(task *asynq.Task, opts []asynq.Option, err error) {
					ignored := []error{errDiscardMessage}
					if a.locker == nil {
						ignored = append(ignored, asynq.ErrDuplicateTask, asynq.ErrTaskIDConflict)
					}
					if err = utils.ErrIgnore(err, ignored...); err == nil {
						return
					}
					taskName := "unknown"
					if task != nil {
						taskName = a.unformatTaskName(task.Type())
					}
					a.warn(ctx, "enqueue task %s failed: %s", taskName, err)
				},
			},
			SyncInterval: utils.Must(time.ParseDuration(a.c.RefreshTasksInterval)),
		}),
	)
	a.id = a.trigger.ID()
}

func (a *asynqRouter) initServer(ctx context.Context, wrapper *asynqWrapper, logLevel asynq.LogLevel) {
	a.ServeMux = asynq.NewServeMux()
	for pattern, taskCfg := range a.c.Tasks {
		if utils.IsStrBlank(taskCfg.Callback) {
			continue
		}
		handler := *(*routerHandleFunc)(inspect.FuncOf(taskCfg.Callback))
		a.ServeMux.Handle(a.formatTaskName(pattern), a.adaptAsynqHandlerFunc(handler))
	}

	asynqCfg := asynq.Config{
		Concurrency:    a.c.ServerConcurrency,
		BaseContext:    context.Background,
		RetryDelayFunc: asynq.DefaultRetryDelayFunc,
		IsFailure:      func(err error) bool { return !errors.Is(err, errDiscardMessage) },
		Queues:         nil,
		StrictPriority: false,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			taskName := "unknown"
			if task != nil {
				taskName = a.unformatTaskName(task.Type())
			}
			a.info(ctx, "handle task %s message error %s", taskName, err)
		}),
		Logger:          a.logger,
		LogLevel:        logLevel,
		ShutdownTimeout: 8 * time.Second,
		HealthCheckFunc: func(err error) {
			if err != nil {
				a.warn(ctx, "health check check failed: %s", err)
			}
		},
		HealthCheckInterval:      15 * time.Second,
		DelayedTaskCheckInterval: 5 * time.Second,
		GroupGracePeriod:         1 * time.Minute,
		GroupMaxDelay:            0,
		GroupMaxSize:             0,
		GroupAggregator:          nil,
		DisableRedisConnClose:    true,
	}
	if utils.IsStrNotBlank(a.c.Queue) {
		asynqCfg.Queues = map[string]int{a.c.Queue: 3}
	}

	a.server = asynq.NewServer(wrapper, asynqCfg)
}

func (a *asynqRouter) preEnqueueFunc(ctx context.Context) func(*asynq.Task, []asynq.Option) error {
	return func(task *asynq.Task, opts []asynq.Option) (err error) {
		// when locker is disabled, we cannot determine which message should be discarded
		if a.locker == nil {
			return
		}

		taskName := a.unformatTaskName(task.Type())
		lockKey := a.formatLockKey(taskName)
		if err = a.locker.Lock(ctx, lockKey, lock.Expire(tolerantOfTimeNotSync), lock.ReentrantKey(a.id)); err == nil {
			a.info(ctx, "pre enqueue task %s success", taskName)
			return
		}

		err = utils.ErrIgnore(err, lock.ErrTimeout, lock.ErrContextDone)
		if err == nil {
			a.debug(ctx, "pre enqueue discard task %s", taskName)
			return errDiscardMessage
		}

		a.warn(ctx, "pre enqueue task %s failed: %s", taskName, err)
		return
	}
}

func (a *asynqRouter) postEnqueueFunc(ctx context.Context) func(info *asynq.TaskInfo, err error) {
	return func(info *asynq.TaskInfo, err error) {
		// release lock
		if a.locker != nil {
			defer routine.Go(a.releaseCronTaskLock, routine.Args(ctx, info), routine.AppName(a.appName))
		}

		ignored := []error{errDiscardMessage}
		if a.locker == nil {
			ignored = append(ignored, asynq.ErrDuplicateTask, asynq.ErrTaskIDConflict)
		}

		if err = utils.ErrIgnore(err, ignored...); err == nil {
			return
		}
		taskName := "unknown"
		if info != nil {
			taskName = a.unformatTaskName(info.Type)
		}
		a.debug(ctx, "post enqueue task %s failed: %s", taskName, err)
	}
}

func (a *asynqRouter) releaseCronTaskLock(ctx context.Context, info *asynq.TaskInfo) {
	if info == nil {
		return
	}
	taskName := a.unformatTaskName(info.Type)

	// 90 ~ 100ms jitter
	jitter := 90*time.Millisecond + time.Duration(float64(10*time.Millisecond)*rand.Float64())

	a.l.RLock()
	lockTime := a.lockDurations[info.Type]
	a.l.RUnlock()

	// prevent a negative tolerant
	tolerant := utils.Min(tolerantOfTimeNotSync, lockTime) - jitter
	tolerant = utils.Max(tolerant, 500*time.Millisecond)
	timer := time.NewTimer(tolerant)
	defer timer.Stop()

	var e error
	defer func() {
		if e != nil {
			a.warn(ctx, "post enqueue task %s release lock failed: %s", taskName, e)
		}
	}()

	now := time.Now()
	unlockKey := a.formatLockKey(taskName)
	for {
		select {
		case <-ctx.Done():
			a.debug(ctx, "post enqueue task %s release lock: context done", taskName)
			e = a.locker.Unlock(ctx, unlockKey, lock.ReentrantKey(a.id))
			return
		case <-timer.C:
			e = a.locker.Unlock(ctx, unlockKey, lock.ReentrantKey(a.id))
			return
		default:
			a.l.RLock()
			newLockTime := a.lockDurations[info.Type]
			a.l.RUnlock()
			if newLockTime != lockTime {
				lockTime = newLockTime
				tolerant = utils.Min(tolerantOfTimeNotSync, lockTime) - jitter
				tolerant = utils.Max(tolerant, 500*time.Millisecond)
				tolerant = utils.Max(0, tolerant-time.Since(now))
				timer.Reset(tolerant)
			}
		}
	}
}

func (a *asynqRouter) gatewayMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, raw *asynq.Task) (err error) {
		taskName := a.unformatTaskName(raw.Type())
		inspect.SetField(raw, asyncqTaskTypenameField, taskName)
		if utils.IsStrBlank(fmkCtx.GetTraceID(ctx)) {
			ctx = fmkCtx.SetTraceID(ctx, utils.NginxID())
		}
		if utils.IsStrBlank(fmkCtx.GetCronTaskName(ctx)) {
			ctx = fmkCtx.SetCronTaskName(ctx, taskName)
		}
		return next.ProcessTask(ctx, raw)
	})
}

func (a *asynqRouter) adaptMiddleware(mw routerMiddleware) asynq.MiddlewareFunc {
	return func(asynqNext asynq.Handler) asynq.Handler {
		next := mw(a.adaptRouterHandlerFunc(asynqNext))
		return a.adaptAsynqHandlerFunc(next)
	}
}

// adaptAsynqHandlerFunc support function signature
// - func(ctx context.Context)
// - func(ctx context.Context) error
// - func(ctx context.Context, args json.Serializable)
// - func(ctx context.Context, args *json.Serializable) error
func (a *asynqRouter) adaptAsynqHandlerFunc(h any) asynq.HandlerFunc {
	if fn, ok := h.(routerHandleFunc); ok {
		return func(ctx context.Context, raw *asynq.Task) (err error) {
			return fn(ctx, a.newTask(raw))
		}
	}
	if fn, ok := h.(func(ctx context.Context, task Task) (err error)); ok {
		return func(ctx context.Context, raw *asynq.Task) (err error) {
			return fn(ctx, a.newTask(raw))
		}
	}

	var (
		hasArg          bool
		argType         reflect.Type
		argTypePtrDepth int
	)
	if reflect.TypeOf(h).NumIn() > 1 {
		argType = reflect.TypeOf(h).In(1)
		for argType.Kind() == reflect.Ptr {
			argType = argType.Elem()
			argTypePtrDepth++
		}
		hasArg = true
	}

	fn := utils.WrapFunc1[error](h)
	return func(ctx context.Context, raw *asynq.Task) (err error) {
		if !hasArg {
			return fn(ctx)
		}
		arg := reflect.New(argType)
		payload := raw.Payload()
		if len(payload) == 0 {
			payload = []byte("null")
		}
		if err = json.Unmarshal(payload, arg.Interface()); err != nil {
			return
		}
		arg = arg.Elem()
		for i := 0; i < argTypePtrDepth; i++ {
			arg = arg.Addr()
		}

		return fn(ctx, arg.Interface())
	}
}

func (a *asynqRouter) adaptRouterHandlerFunc(h asynq.Handler) routerHandleFunc {
	return func(ctx context.Context, raw Task) (err error) {
		return h.ProcessTask(ctx, a.newAsynqTask(raw))
	}
}

func (a *asynqRouter) defaultQueue() (result string) {
	return fmt.Sprintf("%s:cron", config.Use(a.appName).AppName())
}
func (a *asynqRouter) formatLockKey(taskName string) string {
	return fmt.Sprintf("cron_%s", taskName)
}
func (a *asynqRouter) formatTaskName(taskName string) (result string) {
	return fmt.Sprintf("%s:cron:%s", config.Use(a.appName).AppName(), taskName)
}
func (a *asynqRouter) unformatTaskName(taskName string) (result string) {
	return strings.TrimPrefix(taskName, fmt.Sprintf("%s:cron:", config.Use(a.appName).AppName()))
}

func (a *asynqRouter) newTask(raw *asynq.Task) (t Task) {
	return &task{
		id:         raw.Type(),
		name:       raw.Type(),
		payload:    raw.Payload(),
		rawMessage: raw,
	}
}

func (a *asynqRouter) newAsynqTask(raw Task) (t *asynq.Task) {
	return raw.RawMessage().(*asynq.Task)
}

type asynqWrapper struct {
	appName string

	r        *asynqRouter
	n        string
	cli      rdsDrv.UniversalClient
	provider asynq.PeriodicTaskConfigProvider
}

func (a *asynqWrapper) MakeRedisClient() any {
	return a.cli
}

func (a *asynqWrapper) GetConfigs() (result []*asynq.PeriodicTaskConfig, err error) {
	result, err = a.getConfigs()
	if err != nil {
		return
	}

	a.r.l.Lock()
	defer a.r.l.Unlock()
	for _, cfg := range result {
		// renaming
		taskName := inspect.GetField[string](cfg.Task, asyncqTaskTypenameField)
		inspect.SetField(cfg.Task, asyncqTaskTypenameField, a.r.formatTaskName(taskName))

		name := cfg.Task.Type()
		a.r.lockDurations[name], err = a.getTaskExecuteInterval(cfg.Cronspec)
		if err != nil {
			return
		}
	}

	return
}

func (a *asynqWrapper) getConfigs() (result []*asynq.PeriodicTaskConfig, err error) {
	if a.provider != nil {
		result, err = a.provider.GetConfigs()
		if err != nil {
			return
		}
	}

	var confs map[string]*Conf
	if err = config.Use(a.appName).LoadComponentConfig(config.ComponentCron, &confs); err != nil {
		return
	}
	conf, ok := confs[a.n]
	if !ok {
		return nil, errors.Errorf("%s cron config not found", a.n)
	}

	loc, _ := time.LoadLocation(a.r.c.Timezone)
	if loc == nil {
		loc = constant.DefaultLocation()
	}

	queue := conf.Queue
	if utils.IsStrBlank(queue) {
		queue = a.r.c.Queue
	}
	for name, cfg := range conf.Tasks {
		var (
			deadline          time.Time
			interval, timeout time.Duration
			opts              []asynq.Option
		)
		if interval, err = a.getTaskExecuteInterval(cfg.Crontab); err != nil {
			return
		}
		if utils.IsStrNotBlank(cfg.Timeout) {
			if timeout, err = time.ParseDuration(cfg.Timeout); err != nil {
				return
			}
			opts = append(opts, asynq.Timeout(timeout))
		} else {
			opts = append(opts, asynq.Timeout(interval))
		}
		if utils.IsStrNotBlank(cfg.Deadline) {
			if deadline, err = time.ParseInLocation(constant.StdTimeLayout, cfg.Deadline, loc); err != nil {
				return
			}
			opts = append(opts, asynq.Deadline(deadline))
		}

		result = append(result, &asynq.PeriodicTaskConfig{
			Cronspec: cfg.Crontab,
			Task:     asynq.NewTask(name, []byte(cfg.Payload)),
			Opts: append(opts, []asynq.Option{
				asynq.TaskID(name),
				asynq.Unique(utils.Min(interval, tolerantOfTimeNotSync)),
				asynq.Queue(queue),
				asynq.MaxRetry(utils.Max(0, cfg.Retry)),
			}...),
		})
	}
	return
}

func (a *asynqWrapper) getTaskExecuteInterval(spec string) (interval time.Duration, err error) {
	now := time.Now()
	scheduler, err := cron.ParseStandard(spec)
	if err != nil {
		return 0, err
	}
	next := scheduler.Next(now)
	interval = scheduler.Next(next).Sub(next)
	return
}

func init() {
	rand.Seed(time.Now().UnixMicro())
}
