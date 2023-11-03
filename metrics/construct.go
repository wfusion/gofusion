package metrics

import (
	"context"
	"log"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
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
		if instances != nil {
			for _, sinks := range instances[opt.AppName] {
				for name, sink := range sinks {
					log.Printf("%v [Gofusion] %s %s %s router exiting...",
						pid, app, config.ComponentMetrics, name)
					sink.shutdown()
					log.Printf("%v [Gofusion] %s %s %s router exited",
						pid, app, config.ComponentMetrics, name)
				}
			}
			delete(instances, opt.AppName)
			delete(cfgsMap, opt.AppName)
		}
	}
}

func addConfig(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	interval, err := time.ParseDuration(conf.Interval)
	if err != nil {
		panic(errors.Errorf("metrics component parse %s interval failed: %s", name, err))
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

	cfgsMap[opt.AppName][name] = &cfg{
		c:          conf,
		ctx:        ctx,
		name:       name,
		appName:    opt.AppName,
		interval:   interval,
		initOption: opt,
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
	if instances == nil {
		instances = make(map[string]map[string]map[string]Sink)
	}
	appInstances, ok := instances[conf.appName]
	if !ok {
		appInstances = make(map[string]map[string]Sink)
		instances[conf.appName] = appInstances
	}

	jobs, ok := appInstances[conf.name]
	if !ok {
		jobs = make(map[string]Sink)
		appInstances[conf.name] = jobs
	}
	sink, ok = jobs[job]
	if ok {
		return
	}

	switch conf.c.Type {
	case metricsTypePrometheus:
		switch conf.c.Mode {
		case modePull:
			sink = newPrometheusPull(conf.ctx, conf.appName, conf.name, job, conf.c)
		case modePush:
			sink = newPrometheusPush(conf.ctx, conf.appName, conf.name, job, conf.interval, conf.c)
		}
	default:
		panic(errors.Errorf("unknown metrics type: %s", conf.c.Type))
	}

	if sink == nil {
		panic(errors.Errorf("unknown metrics mode: %s", conf.c.Mode))
	}

	jobs[job] = sink
	return
}

func init() {
	config.AddComponent(config.ComponentMetrics, Construct)
}
