package metrics

import (
	"context"
	"log"
	"reflect"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/metrics"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"

	fusLog "github.com/wfusion/gofusion/log"

	_ "github.com/wfusion/gofusion/log/customlogger"
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}
	for name, conf := range confs {
		addConfig(ctx, name, conf, opt)
	}

	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if appInstances != nil {
			for _, sinks := range appInstances[opt.AppName] {
				for name, sink := range sinks {
					log.Printf("%v [Gofusion] %s %s %s router exiting...",
						pid, app, config.ComponentMetrics, name)
					sink.shutdown()
					log.Printf("%v [Gofusion] %s %s %s router exited",
						pid, app, config.ComponentMetrics, name)
				}
			}
			delete(appInstances, opt.AppName)
		}

		if cfgsMap != nil {
			delete(cfgsMap, opt.AppName)
		}
	}
}

func addConfig(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var (
		err      error
		interval time.Duration
	)
	if utils.IsStrNotBlank(conf.Interval) {
		interval, err = time.ParseDuration(conf.Interval)
		if err != nil {
			panic(errors.Errorf("metrics component parse %s interval failed: %s", name, err))
		}
	}

	rwlock.Lock()
	defer rwlock.Unlock()
	if cfgsMap == nil {
		cfgsMap = make(map[string]map[string]*cfg)
	}
	if cfgsMap[opt.AppName] == nil {
		cfgsMap[opt.AppName] = make(map[string]*cfg)
	}
	if _, ok := cfgsMap[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}

	var logger metrics.Logger
	if utils.IsStrNotBlank(conf.Logger) {
		loggerType := inspect.TypeOf(conf.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			logger := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
			loggerValue.Interface().(customLogger).Init(logger, opt.AppName, name)
		}
		logger = loggerValue.Convert(metricsLoggerType).Interface().(metrics.Logger)
	}

	cfgsMap[opt.AppName][name] = &cfg{
		c:          conf,
		ctx:        ctx,
		name:       name,
		appName:    opt.AppName,
		interval:   interval,
		initOption: opt,
		logger:     logger,
	}
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func NewDI(name, job string, opts ...utils.OptionExtender) func() Sink {
	return func() Sink {
		return Use(name, job, opts...)
	}
}

func Use(name, job string, opts ...utils.OptionExtender) Sink {
	opt := utils.ApplyOptions[useOption](opts...)
	rwlock.Lock()
	defer rwlock.Unlock()
	cfgs, ok := cfgsMap[opt.appName]
	if !ok {
		panic(errors.Errorf("app metrics config not found: %s", opt.appName))
	}
	cfg, ok := cfgs[name]
	if !ok {
		panic(errors.Errorf("metrics config not found: %s", name))
	}

	return use(job, cfg)
}

func use(job string, conf *cfg) (sink Sink) {
	if appInstances == nil {
		appInstances = make(map[string]map[string]map[string]Sink)
	}
	instances, ok := appInstances[conf.appName]
	if !ok {
		instances = make(map[string]map[string]Sink)
		appInstances[conf.appName] = instances
	}

	jobs, ok := instances[conf.name]
	if !ok {
		jobs = make(map[string]Sink)
		instances[conf.name] = jobs
	}
	sink, ok = jobs[job]
	if ok {
		return
	}

	switch conf.c.Type {
	case metricsTypePrometheus:
		switch conf.c.Mode {
		case modePull:
			sink = newPrometheusPull(conf.ctx, conf.appName, conf.name, job, conf)
		case modePush:
			sink = newPrometheusPush(conf.ctx, conf.appName, conf.name, job, conf.interval, conf)
		}
	case metricsTypeMock:
		sink = newMock(conf.ctx, conf.appName, conf.name, job, conf)
	default:
		panic(errors.Errorf("unknown metrics type: %s", conf.c.Type))
	}

	if sink == nil {
		panic(errors.Errorf("unknown metrics mode: %s", conf.c.Mode))
	}

	jobs[job] = sink
	return
}

func Internal(opts ...utils.OptionExtender) (sinks []Sink) {
	opt := utils.ApplyOptions[useOption](opts...)
	appName := config.Use(opt.appName).AppName()
	rwlock.Lock()
	defer rwlock.Unlock()
	cfgs, ok := cfgsMap[opt.appName]
	if !ok {
		return
	}
	for _, cfg := range cfgs {
		if cfg.c.EnableInternalMetrics {
			sinks = append(sinks, use(appName, cfg))
		}
	}
	return
}

func init() {
	config.AddComponent(config.ComponentMetrics, Construct, config.WithFlag(&flagString))
}
