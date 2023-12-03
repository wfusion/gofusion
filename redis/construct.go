package redis

import (
	"context"
	"log"
	"reflect"
	"syscall"

	rdsDrv "github.com/redis/go-redis/v9"
	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/infra/drivers/redis"
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
		addInstance(ctx, name, conf, opt)
	}
	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if appInstances != nil {
			for name, instance := range appInstances[opt.AppName] {
				if err := instance.GetProxy().Close(); err != nil {
					log.Printf("%v [Gofusion] %s %s %s close error: %s",
						pid, app, config.ComponentRedis, name, err)
				}
			}
			delete(appInstances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var hooks []rdsDrv.Hook
	for _, hookLoc := range conf.Hooks {
		if hookType := inspect.TypeOf(hookLoc); hookType != nil {
			hookValue := reflect.New(hookType)
			if hookValue.Type().Implements(customLoggerType) {
				logger := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
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
	if appInstances == nil {
		appInstances = make(map[string]map[string]*instance)
	}
	if appInstances[opt.AppName] == nil {
		appInstances[opt.AppName] = make(map[string]*instance)
	}
	if _, ok := appInstances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	appInstances[opt.AppName][name] = &instance{name: name, redis: rdsCli}

	if opt.DI != nil {
		opt.DI.MustProvide(func() rdsDrv.UniversalClient { return Use(ctx, name, AppName(opt.AppName)) }, di.Name(name))
	}

	go startDaemonRoutines(ctx, opt.AppName, name, conf)
}

func init() {
	config.AddComponent(config.ComponentRedis, Construct, config.WithFlag(&flagString))
}
