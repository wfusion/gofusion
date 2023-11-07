package cron

import (
	"context"
	"log"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/utils"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/config"
)

var (
	locker  sync.RWMutex
	routers map[string]map[string]IRouter
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}

	for name, conf := range confs {
		addInstance(ctx, name, conf, opt)
	}
	return func() {
		locker.Lock()
		defer locker.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if routers != nil {
			for name, router := range routers[opt.AppName] {
				log.Printf("%v [Gofusion] %s %s %s exiting...", pid, app, config.ComponentCron, name)
				if err := router.shutdown(); err == nil {
					log.Printf("%v [Gofusion] %s %s %s exited", pid, app, config.ComponentCron, name)
				} else {
					log.Printf("%v [Gofusion] %s %s %s exit failed: %s", pid, app, config.ComponentCron, name, err)
				}
				delete(routers[opt.AppName], name)
			}
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var r IRouter
	switch conf.Type {
	case schedulerTypeAsynq:
		r = newAsynq(ctx, opt.AppName, name, conf)
	default:
		panic(ErrUnsupportedSchedulerType)
	}

	locker.Lock()
	defer locker.Unlock()
	if routers == nil {
		routers = make(map[string]map[string]IRouter)
	}
	if routers[opt.AppName] == nil {
		routers[opt.AppName] = make(map[string]IRouter)
	}
	if _, ok := routers[opt.AppName][name]; ok {
		panic(errors.Errorf("duplicated cron name: %s", name))
	}
	routers[opt.AppName][name] = r

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() IRouter { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
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

func Use(name string, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	routers, ok := routers[opt.appName]
	if !ok {
		panic(errors.Errorf("cron router instance not found for app: %s", opt.appName))
	}

	router, ok := routers[name]
	if !ok {
		panic(errors.Errorf("cron router instance not found for name: %s", name))
	}
	return router
}

func init() {
	config.AddComponent(config.ComponentCron, Construct)
}
