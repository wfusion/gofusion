package kv

import (
	"context"
	"log"
	"syscall"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func(context.Context) {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}
	for name, conf := range confs {
		addInstance(ctx, name, conf, opt)
	}

	return func(context.Context) {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if appInstances != nil {
			for name, instance := range appInstances[opt.AppName] {
				if err := instance.close(); err != nil {
					log.Printf("%v [Gofusion] %s %s %s close error: %s",
						pid, app, config.ComponentKV, name, err)
				}
			}
			delete(appInstances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var instance Storable
	switch conf.Type {
	case kvTypeRedis:
		instance = newRedisInstance(ctx, name, conf, opt)
	case kvTypeConsul:
		instance = newConsulInstance(ctx, name, conf, opt)
	case kvTypeEtcd:
		instance = newEtcdInstance(ctx, name, conf, opt)
	case kvTypeZK:
		instance = newZKInstance(ctx, name, conf, opt)
	default:
		panic(ErrUnsupportedKVType)
	}

	rwlock.Lock()
	defer rwlock.Unlock()
	if appInstances == nil {
		appInstances = make(map[string]map[string]Storable)
	}
	if appInstances[opt.AppName] == nil {
		appInstances[opt.AppName] = make(map[string]Storable)
	}
	if _, ok := appInstances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	appInstances[opt.AppName][name] = instance

	if opt.DI != nil {
		opt.DI.MustProvide(func() Storable { return Use(name, AppName(opt.AppName)) }, di.Name(name))
	}
	if opt.App != nil {
		opt.App.MustProvide(
			func() Storable { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}

	go startDaemonRoutines(ctx, opt.AppName, name, conf)
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func NewDI(name string, opts ...utils.OptionExtender) func() Storable {
	return func() Storable {
		return Use(name, opts...)
	}
}

func Use(name string, opts ...utils.OptionExtender) Storable {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := appInstances[opt.appName]
	if !ok {
		panic(errors.Errorf("kv instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("kv instance not found for name: %s", name))
	}
	return instance
}

func init() {
	config.AddComponent(config.ComponentKV, Construct, config.WithFlag(&flagString))
}
