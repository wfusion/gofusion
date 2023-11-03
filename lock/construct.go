package lock

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/redis"
)

var (
	instances map[string]map[string]Lockable
	rwlock    sync.RWMutex
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
		rwlock.Lock()
		defer rwlock.Unlock()
		if instances != nil {
			delete(instances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	rwlock.Lock()
	defer rwlock.Unlock()
	if instances == nil {
		instances = make(map[string]map[string]Lockable)
	}
	if instances[opt.AppName] == nil {
		instances[opt.AppName] = make(map[string]Lockable)
	}

	if _, ok := instances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}

	switch conf.Type {
	case lockTypeRedisLua:
		redis.Use(ctx, conf.Instance, redis.AppName(opt.AppName)) // check if instance exists
		instances[opt.AppName][name] = newRedisLuaLocker(ctx, opt.AppName, conf.Instance)
	case lockTypeRedisNX:
		redis.Use(ctx, conf.Instance, redis.AppName(opt.AppName)) // check if instance exists
		instances[opt.AppName][name] = newRedisNXLocker(ctx, opt.AppName, conf.Instance)
	case lockTypeMySQL:
		db.Use(ctx, conf.Instance, db.AppName(opt.AppName)) // check if instance exists
		instances[opt.AppName][name] = newMysqlLocker(ctx, opt.AppName, conf.Instance)
	case lockTypeMariaDB:
		db.Use(ctx, conf.Instance, db.AppName(opt.AppName)) // check if instance exists
		instances[opt.AppName][name] = newMysqlLocker(ctx, opt.AppName, conf.Instance)
	default:
		panic(ErrUnsupportedLockType)
	}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(func() Lockable { return Use(name, AppName(opt.AppName)) }, di.Name(name))
		if _, ok := instances[opt.AppName][name].(ReentrantLockable); ok {
			opt.DI.MustProvide(
				func() ReentrantLockable { return UseReentrant(ctx, name, AppName(opt.AppName)) },
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

func Use(name string, opts ...utils.OptionExtender) Lockable {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := instances[opt.appName]
	if !ok {
		panic(errors.Errorf("locker instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("locker instance not found for name: %s", name))
	}
	return instance
}

func UseReentrant(ctx context.Context, name string, opts ...utils.OptionExtender) ReentrantLockable {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := instances[opt.appName]
	if !ok {
		panic(errors.Errorf("reentrant locker instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("reentrant locker instance not found for name: %s", name))
	}
	lockable, ok := instance.(ReentrantLockable)
	if !ok {
		panic(errors.Errorf("locker instance is not reentrantable: %s", name))
	}

	return lockable
}

func init() {
	config.AddComponent(config.ComponentLock, Construct)
}
