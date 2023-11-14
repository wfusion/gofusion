package mq

import (
	"context"
	"log"
	"reflect"
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"

	fusLog "github.com/wfusion/gofusion/log"

	_ "github.com/wfusion/gofusion/log/customlogger"
)

var (
	locker      sync.RWMutex
	subscribers = map[string]map[string]Subscriber{}
	publishers  = map[string]map[string]Publisher{}
	routers     = map[string]map[string]IRouter{}
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
				log.Printf("%v [Gofusion] %s %s %s router exiting...",
					pid, app, config.ComponentMessageQueue, name)
				if err := router.close(); err == nil {
					log.Printf("%v [Gofusion] %s %s %s router exited",
						pid, app, config.ComponentMessageQueue, name)
				} else {
					log.Printf("%v [Gofusion] %s %s %s router exit failed: %s",
						pid, app, config.ComponentMessageQueue, name, err)
				}
			}
			delete(routers, opt.AppName)
		}

		if publishers != nil {
			for name, publisher := range publishers[opt.AppName] {
				log.Printf("%v [Gofusion] %s %s %s publisher exiting...",
					pid, app, config.ComponentMessageQueue, name)
				if err := publisher.close(); err == nil {
					log.Printf("%v [Gofusion] %s %s %s publisher exited",
						pid, app, config.ComponentMessageQueue, name)
				} else {
					log.Printf("%v [Gofusion] %s %s %s publisher exit failed: %s",
						pid, app, config.ComponentMessageQueue, name, err)
				}
			}
			delete(publishers, opt.AppName)
		}

		if subscribers != nil {
			for name, subscriber := range subscribers[opt.AppName] {
				log.Printf("%v [Gofusion] %s %s %s subscriber exiting...",
					pid, app, config.ComponentMessageQueue, name)
				if err := subscriber.close(); err == nil {
					log.Printf("%v [Gofusion] %s %s %s subscriber exited",
						pid, app, config.ComponentMessageQueue, name)
				} else {
					log.Printf("%v [Gofusion] %s %s %s subscriber exit failed: %s",
						pid, app, config.ComponentMessageQueue, name, err)
				}
			}
			delete(subscribers, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var logger watermill.LoggerAdapter
	if utils.IsStrNotBlank(conf.Logger) {
		loggerType := inspect.TypeOf(conf.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			l := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
			loggerValue.Interface().(customLogger).Init(l, opt.AppName, name)
		}
		logger = loggerValue.Convert(watermillLoggerType).Interface().(watermill.LoggerAdapter)
	}

	if conf.ConsumerConcurrency < 1 {
		conf.ConsumerConcurrency = 1
	}

	var (
		puber Publisher
		suber Subscriber
	)
	newFunc, ok := newFn[conf.Type]
	if ok {
		puber, suber = newFunc(ctx, opt.AppName, name, conf, logger)
	} else {
		panic(errors.Errorf("unknown message queue type: %+v", conf.Type))
	}

	locker.Lock()
	defer locker.Unlock()
	if suber != nil {
		if subscribers == nil {
			subscribers = make(map[string]map[string]Subscriber)
		}
		if subscribers[opt.AppName] == nil {
			subscribers[opt.AppName] = make(map[string]Subscriber)
		}
		if _, ok := subscribers[name]; ok {
			panic(ErrDuplicatedSubscriberName)
		}
		subscribers[opt.AppName][name] = suber

		if routers == nil {
			routers = make(map[string]map[string]IRouter)
		}
		if routers[opt.AppName] == nil {
			routers[opt.AppName] = make(map[string]IRouter)
		}
		if _, ok := routers[opt.AppName][name]; ok {
			panic(ErrDuplicatedRouterName)
		}
		routers[opt.AppName][name] = newRouter(ctx, opt.AppName, name, conf, puber, suber, logger)

		// ioc
		if opt.DI != nil {
			opt.DI.
				MustProvide(func() Subscriber { return sub(name, AppName(opt.AppName)) }, di.Name(name)).
				MustProvide(func() IRouter { return Use(name, AppName(opt.AppName)) }, di.Name(name))
		}

	}

	if puber != nil {
		if publishers == nil {
			publishers = make(map[string]map[string]Publisher)
		}
		if publishers[opt.AppName] == nil {
			publishers[opt.AppName] = make(map[string]Publisher)
		}
		if _, ok := publishers[opt.AppName][name]; ok {
			panic(ErrDuplicatedPublisherName)
		}
		publishers[opt.AppName][name] = puber

		// ioc
		if opt.DI != nil {
			opt.DI.MustProvide(func() Publisher { return Pub(name, AppName(opt.AppName)) }, di.Name(name))
		}
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

func sub(name string, opts ...utils.OptionExtender) Subscriber {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	subscribers, ok := subscribers[opt.appName]
	if !ok {
		panic(errors.Errorf("mq subscriber instance not found for app: %s", opt.appName))
	}
	subscriber, ok := subscribers[name]
	if !ok {
		panic(errors.Errorf("mq subscriber instance not found for name: %s", name))
	}
	return subscriber
}

func Pub(name string, opts ...utils.OptionExtender) Publisher {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	publishers, ok := publishers[opt.appName]
	if !ok {
		panic(errors.Errorf("mq publisher instance not found for app: %s", opt.appName))
	}
	publisher, ok := publishers[name]
	if !ok {
		panic(errors.Errorf("mq publisher instance not found for name: %s", name))
	}
	return publisher
}

func Sub(name string, opts ...utils.OptionExtender) Subscriber {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	subscribers, ok := subscribers[opt.appName]
	if !ok {
		panic(errors.Errorf("mq subscriber instance not found for app: %s", opt.appName))
	}
	subscriber, ok := subscribers[name]
	if !ok {
		panic(errors.Errorf("mq subscriber instance not found for name: %s", name))
	}
	return subscriber
}

func Use(name string, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	routers, ok := routers[opt.appName]
	if !ok {
		panic(errors.Errorf("mq router instance not found for app: %s", opt.appName))
	}
	r, ok := routers[name]
	if !ok {
		panic(errors.Errorf("mq router instance not found for name: %s", name))
	}
	return r
}

func init() {
	config.AddComponent(config.ComponentMessageQueue, Construct)
}
