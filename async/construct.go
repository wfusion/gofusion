package async

import (
	"context"
	"log"
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"

	_ "github.com/wfusion/gofusion/log/customlogger"
)

var (
	locker    sync.RWMutex
	consumers = map[string]map[string]Consumable{}
	producers = map[string]map[string]Producable{}
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
		if consumers != nil {
			for name, router := range consumers[opt.AppName] {
				log.Printf("%v [Gofusion] %s %s %s exiting...", pid, app, config.ComponentAsync, name)
				if err := router.shutdown(); err == nil {
					log.Printf("%v [Gofusion] %s %s %s exited", pid, app, config.ComponentAsync, name)
				} else {
					log.Printf("%v [Gofusion] %s %s %s exit failed: %s", pid, app, config.ComponentAsync, name, err)
				}
			}
			delete(consumers, opt.AppName)
		}

		if producers != nil {
			producers[opt.AppName] = make(map[string]Producable, len(producers))
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var (
		producer Producable
		consumer Consumable
	)
	switch conf.Type {
	case asyncTypeAsynq:
		if conf.Producer {
			producer = newAsynqProducer(ctx, opt.AppName, name, conf)
		}
		if conf.Consumer {
			consumer = newAsynqConsumer(ctx, opt.AppName, name, conf)
		}
	case asyncTypeMysql:
		fallthrough
	default:
		panic(ErrUnsupportedSchedulerType)
	}

	locker.Lock()
	defer locker.Unlock()
	if consumer != nil {
		if consumers == nil {
			consumers = make(map[string]map[string]Consumable)
		}
		if consumers[opt.AppName] == nil {
			consumers[opt.AppName] = make(map[string]Consumable)
		}
		if _, ok := consumers[name]; ok {
			panic(ErrDuplicatedInstanceName)
		}
		consumers[opt.AppName][name] = consumer

		// ioc
		if opt.DI != nil {
			opt.DI.MustProvide(
				func() Consumable { return C(name, AppName(opt.AppName)) },
				di.Name(name),
			)
		}
		if opt.App != nil {
			opt.App.MustProvide(
				func() Consumable { return C(name, AppName(opt.AppName)) },
				di.Name(name),
			)
		}
	}

	if producer != nil {
		if producers == nil {
			producers = make(map[string]map[string]Producable)
		}
		if producers[opt.AppName] == nil {
			producers[opt.AppName] = make(map[string]Producable)
		}
		if _, ok := producers[name]; ok {
			panic(ErrDuplicatedInstanceName)
		}
		producers[opt.AppName][name] = producer

		// ioc
		if opt.DI != nil {
			opt.DI.MustProvide(
				func() Producable { return P(name) },
				di.Name(name),
			)
		}
		if opt.App != nil {
			opt.App.MustProvide(
				func() Producable { return P(name) },
				di.Name(name),
			)
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

func C(name string, opts ...utils.OptionExtender) Consumable {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	consumers, ok := consumers[opt.appName]
	if !ok {
		panic(errors.Errorf("async consumer instance not found for app: %s", opt.appName))
	}
	consumer, ok := consumers[name]
	if !ok {
		panic(errors.Errorf("async consumer instance not found for name: %s", name))
	}
	return consumer
}

func P(name string, opts ...utils.OptionExtender) Producable {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	producers, ok := producers[opt.appName]
	if !ok {
		panic(errors.Errorf("async producer instance not found for app: %s", opt.appName))
	}
	producer, ok := producers[name]
	if !ok {
		panic(errors.Errorf("async producer instance not found for name: %s", name))
	}
	return producer
}

func init() {
	config.AddComponent(config.ComponentAsync, Construct, config.WithFlag(&flagString))
}
