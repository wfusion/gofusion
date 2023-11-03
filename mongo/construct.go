package mongo

import (
	"context"
	"log"
	"reflect"
	"syscall"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/infra/drivers/mongo"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/routine"

	mgoEvt "go.mongodb.org/mongo-driver/event"
	mgoDrv "go.mongodb.org/mongo-driver/mongo"

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
				if err := instance.GetProxy().Disconnect(nil); err != nil {
					log.Printf("%v [Gofusion] %s %s disconnect error: %s", pid, app, config.ComponentMongo, err)
				}
			}
			delete(instances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var monitor *mgoEvt.CommandMonitor
	if utils.IsStrNotBlank(conf.LoggerConfig.Logger) {
		loggerType := inspect.TypeOf(conf.LoggerConfig.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			l := fmkLog.Use(conf.LoggerConfig.LogInstance, fmkLog.AppName(opt.AppName))
			loggerValue.Interface().(customLogger).Init(l, opt.AppName, name)
		}
		monitor = loggerValue.Interface().(logger).GetMonitor()
	}

	// conf.Option.Password = config.CryptoDecryptFunc()(conf.Option.Password)
	mgoCli, err := mongo.Default.New(ctx, conf.Option, mongo.WithMonitor(monitor))
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
	instances[opt.AppName][name] = &instance{mongo: mgoCli, name: name, database: conf.DB}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() *mgoDrv.Database {
				return Use(ctx, name, AppName(opt.AppName)).Database
			},
			di.Name(name),
		)
	}

	routine.Loop(startDaemonRoutines, routine.Args(ctx, opt.AppName, name), routine.AppName(opt.AppName))
}

func init() {
	config.AddComponent(config.ComponentMongo, Construct)
}
