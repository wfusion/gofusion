package metrics

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/infra/metrics"
	"github.com/wfusion/gofusion/common/infra/metrics/prometheus"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"
)

const (
	defaultQueueLimit            = 16 * 1024
	defaultMetricsPoolNameFormat = "base:metrics:%s:%s:%s"
)

var (
	rwlock    = new(sync.RWMutex)
	instances map[string]map[string]map[string]Sink
	cfgsMap   map[string]map[string]*cfg
)

type abstract struct {
	*metrics.Metrics

	ctx         context.Context
	job         string
	name        string
	appName     string
	log         log.Logable
	constLabels []metrics.Label

	stop      chan struct{}
	queue     chan *task
	queuePool routine.Pool

	dispatcher map[string]func(...any)
}

type task struct {
	ctx    context.Context
	key    []string
	val    any
	opts   []utils.OptionExtender
	labels []metrics.Label
	method string
}

func (t *task) String() string {
	label := make(map[string]string, len(t.labels))
	for _, l := range t.labels {
		label[l.Name] = l.Value
	}

	return fmt.Sprintf("%s:%s:%+v(%+v)", t.method, strings.Join(t.key, constant.Dot), t.val, label)
}

func newMetrics(ctx context.Context, appName, name, job string, sink metrics.MetricSink, conf *Conf) *abstract {
	metricsConfig := metrics.DefaultConfig(appName)
	if conf.EnableRuntimeMetrics {
		metricsConfig.EnableRuntimeMetrics = true
	} else {
		metricsConfig.EnableRuntimeMetrics = false
	}
	if conf.EnableServiceLabel {
		metricsConfig.EnableHostname = true
		metricsConfig.EnableHostnameLabel = true
		metricsConfig.EnableServiceLabel = true
		metricsConfig.EnableClientIPLabel = true
	} else {
		metricsConfig.EnableHostname = false
		metricsConfig.EnableHostnameLabel = false
		metricsConfig.EnableServiceLabel = false
		metricsConfig.EnableClientIPLabel = false
	}

	m, err := metrics.New(metricsConfig, sink)
	if err != nil {
		panic(errors.Errorf("initialize metrics failed: %s", err))
	}

	var logger log.Logable
	if utils.IsStrNotBlank(conf.LogInstance) {
		logger = log.Use(conf.LogInstance, log.AppName(appName))
	}

	limit := defaultQueueLimit
	if conf.QueueLimit > 0 {
		limit = conf.QueueLimit
	}
	if conf.QueueConcurrency == 0 {
		conf.QueueConcurrency = runtime.NumCPU()
	}

	constLabels := make([]metrics.Label, 0, len(conf.Labels))
	for k, v := range conf.Labels {
		constLabels = append(constLabels, metrics.Label{Name: k, Value: v})
	}

	a := &abstract{
		Metrics: m,

		ctx:         ctx,
		constLabels: constLabels,
		job:         job,
		name:        name,
		appName:     appName,
		log:         logger,
		stop:        make(chan struct{}),
		queue:       make(chan *task, limit),
		queuePool: routine.NewPool(fmt.Sprintf(defaultMetricsPoolNameFormat, appName, name, job), conf.QueueConcurrency,
			routine.AppName(appName), routine.WithoutTimeout()),
	}

	a.dispatcher = map[string]func(...any){
		"Gauge":        utils.WrapFunc(a.setGaugeWithLabels),
		"Counter":      utils.WrapFunc(a.incrCounterWithLabels),
		"Sample":       utils.WrapFunc(a.addSampleWithLabels),
		"MeasureSince": utils.WrapFunc(a.measureSinceWithLabels),
	}

	a.serve()

	return a
}

func (a *abstract) SetGauge(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender) {
	a.send(ctx, "Gauge", key, val, opts...)
}
func (a *abstract) IncrCounter(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender) {
	a.send(ctx, "Counter", key, val, opts...)
}
func (a *abstract) AddSample(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender) {
	a.send(ctx, "Sample", key, val, opts...)
}
func (a *abstract) MeasureSince(ctx context.Context, key []string, start time.Time, opts ...utils.OptionExtender) {
	a.send(ctx, "MeasureSince", key, start, opts...)
}

func (a *abstract) getProxy() any {
	return inspect.GetField[metrics.MetricSink](a.Metrics, "sink")
}
func (a *abstract) shutdown() {
	if _, ok := utils.IsChannelClosed(a.stop); ok {
		return
	}

	a.Metrics.Shutdown()
	close(a.stop)
	close(a.queue)
}

func (a *abstract) send(ctx context.Context, method string, key []string, val any, opts ...utils.OptionExtender) {
	opt := utils.ApplyOptions[option](opts...)

	t := &task{
		ctx:    ctx,
		key:    key,
		val:    val,
		opts:   append(opts, a.convertOpts(opts...)...),
		labels: a.convertLabels(opt.labels),
		method: method,
	}

	switch {
	case opt.timeout > 0:
		timeoutCtx, cancel := context.WithTimeout(a.ctx, opt.timeout)
		defer cancel()

		select {
		case a.queue <- t:
		case <-ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to context done",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-timeoutCtx.Done():
			if a.log != nil {
				a.log.Warn(ctx, "%v [Gofusion] %s %s %s async send task canceled due to timeout %s",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name, opt.timeout)
			}
		case <-a.stop:
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to metrics stopped",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-a.ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to app exited",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		}
	case opt.timeout < 0:
		select {
		case a.queue <- t:
		case <-ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to context done",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-a.stop:
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to metrics stopped",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-a.ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to app exited",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		}
	default:
		select {
		case a.queue <- t:
		case <-ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to context done",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-a.ctx.Done():
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to app exited",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		case <-a.stop:
			if a.log != nil {
				a.log.Info(ctx, "%v [Gofusion] %s %s %s async send task canceled due to metrics stopped",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		default:
			if a.log != nil {
				a.log.Warn(ctx, "%v [Gofusion] %s %s %s async send task canceled due to exceed the queue limit",
					syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
			}
		}
	}
}
func (a *abstract) serve() {
	routine.Loop(func() {
		for {
			select {
			case <-a.ctx.Done():
				if a.log != nil {
					a.log.Info(a.ctx, "%v [Gofusion] %s %s %s process exited due to context done",
						syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
					return
				}
			case task, ok := <-a.queue:
				if !ok {
					a.log.Info(a.ctx, "%v [Gofusion] %s %s %s process exited due to queue closed",
						syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name)
					return
				}

				if err := a.queuePool.Submit(a.process, routine.Args(task)); err != nil && a.log != nil {
					a.log.Error(task.ctx, "%v [Gofusion] %s %s %s submit process %s error: %s",
						syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name, task, err)
				}
			}
		}
	}, routine.AppName(a.appName))
}
func (a *abstract) process(task *task) (err error) {
	_, err = utils.Catch(func() (err error) {
		handler, ok := a.dispatcher[task.method]
		if !ok {
			return errors.Errorf("method %s not found", task.method)
		}
		params := []any{task.key, task.val, append(task.labels, a.constLabels...)}
		params = append(params, utils.SliceMapping(task.opts, func(o utils.OptionExtender) any { return o })...)
		handler(params...)
		return
	})
	if err != nil && a.log != nil {
		a.log.Error(task.ctx, "%v [Gofusion] %s %s %s process %s catch error: %s",
			syscall.Getpid(), config.Use(a.appName).AppName(), config.ComponentMetrics, a.name, task, err)
	}
	return
}

func (a *abstract) setGaugeWithLabels(key []string, v any, labels []metrics.Label, opts ...utils.OptionExtender) {
	val, ok := v.(float64)
	if !ok {
		return
	}
	opt := utils.ApplyOptions[option](opts...)
	if opt.precision {
		a.Metrics.SetPrecisionGaugeWithLabels(key, val, labels, opts...)
	} else {
		a.Metrics.SetGaugeWithLabels(key, float32(val), labels, opts...)
	}
}
func (a *abstract) incrCounterWithLabels(key []string, v any, labels []metrics.Label, opts ...utils.OptionExtender) {
	val, ok := v.(float64)
	if !ok {
		return
	}
	a.Metrics.IncrCounterWithLabels(key, float32(val), labels, opts...)
}
func (a *abstract) addSampleWithLabels(key []string, v any, labels []metrics.Label, opts ...utils.OptionExtender) {
	val, ok := v.(float64)
	if !ok {
		return
	}
	opt := utils.ApplyOptions[option](opts...)
	if opt.precision {
		a.Metrics.AddPrecisionSampleWithLabels(key, val, labels, opts...)
	} else {
		a.Metrics.AddSampleWithLabels(key, float32(val), labels, opts...)
	}
}
func (a *abstract) measureSinceWithLabels(key []string, v any, labels []metrics.Label, opts ...utils.OptionExtender) {
	start, ok := v.(time.Time)
	if !ok {
		return
	}
	a.Metrics.MeasureSinceWithLabels(key, start, labels, opts...)
}

func (a *abstract) convertLabels(src []Label) (dst []metrics.Label) {
	return utils.SliceMapping(src, func(l Label) metrics.Label {
		return metrics.Label{
			Name:  l.Key,
			Value: l.Value,
		}
	})
}
func (a *abstract) convertOpts(src ...utils.OptionExtender) (dst []utils.OptionExtender) {
	if src == nil {
		return
	}

	dst = make([]utils.OptionExtender, 0, len(src))
	opt := utils.ApplyOptions[option](src...)
	if opt.precision {
		dst = append(dst, metrics.Precision())
	}
	if len(opt.prometheusBuckets) > 0 {
		dst = append(dst, prometheus.Bucket(opt.prometheusBuckets))
	}

	return
}
