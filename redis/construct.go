package redis

import (
	"context"
	"log"
	"reflect"
	"syscall"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/infra/drivers/redis"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/routine"

	rdsDrv "github.com/redis/go-redis/v9"

	fmkLog "github.com/wfusion/gofusion/log"

	_ "github.com/wfusion/gofusion/log/customlogger"
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

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if instances != nil {
			for _, instance := range instances[opt.AppName] {
				if err := instance.GetProxy().Close(); err != nil {
					log.Printf("%v [Gofusion] %s %s close error: %s", pid, app, config.ComponentRedis, err)
				}
			}
			delete(instances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var hooks []rdsDrv.Hook
	for _, hookLoc := range conf.Hooks {
		if hookType := inspect.TypeOf(hookLoc); hookType != nil {
			hookValue := reflect.New(hookType)
			if hookValue.Type().Implements(customLoggerType) {
				logger := fmkLog.Use(conf.LogInstance, fmkLog.AppName(opt.AppName))
				hookValue.Interface().(customLogger).Init(logger, opt.AppName, name)
			}

			hooks = append(hooks, hookValue.Interface().(rdsDrv.Hook))
		}
	}

	// conf.Option.Password = config.CryptoDecryptFunc()(conf.Option.Password)
	rdsCli, err := redis.Default.New(ctx, conf.Option, redis.WithHook(hooks))
	if err != nil {
		panic(err)
	}

	rwlock.Lock()
	defer rwlock.Unlock()
	if instances == nil {
		instances = make(map[string]map[string]*instance)
	}
	if instances[opt.AppName] == nil {
		instances[opt.AppName] = make(map[string]*instance)
	}
	if _, ok := instances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	instances[opt.AppName][name] = &instance{name: name, redis: rdsCli}

	if opt.DI != nil {
		opt.DI.MustProvide(func() rdsDrv.UniversalClient { return Use(ctx, name, AppName(opt.AppName)) }, di.Name(name))
	}

	routine.Loop(startDaemonRoutines, routine.Args(ctx, opt.AppName, name), routine.AppName(opt.AppName))
}

func init() {
	config.AddComponent(config.ComponentRedis, Construct)
}
