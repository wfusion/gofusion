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

	mgoEvt "go.mongodb.org/mongo-driver/event"
	mgoDrv "go.mongodb.org/mongo-driver/mongo"

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
				if err := instance.GetProxy().Disconnect(nil); err != nil {
					log.Printf("%v [Gofusion] %s %s %s disconnect error: %s",
						pid, app, config.ComponentMongo, name, err)
				}
			}
			delete(appInstances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var monitor *mgoEvt.CommandMonitor
	if utils.IsStrNotBlank(conf.LoggerConfig.Logger) {
		loggerType := inspect.TypeOf(conf.LoggerConfig.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			l := fusLog.Use(conf.LoggerConfig.LogInstance, fusLog.AppName(opt.AppName))
			loggerValue.Interface().(customLogger).Init(l, opt.AppName, name)
		}
		monitor = loggerValue.Interface().(logger).GetMonitor()
	}

	// conf.Option.Password = config.CryptoDecryptFunc()(conf.Option.Password)
	mgoCli, err := mongo.Default.New(ctx, conf.Option,
		mongo.WithMonitor(monitor),
		mongo.WithPoolMonitor(&mgoEvt.PoolMonitor{Event: metricsPoolMonitor(opt.AppName, name)}))
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
	appInstances[opt.AppName][name] = &instance{mongo: mgoCli, name: name, database: conf.DB}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() *mgoDrv.Database {
				return Use(name, AppName(opt.AppName)).Database
			},
			di.Name(name),
		)
	}
	if opt.App != nil {
		opt.App.MustProvide(
			func() *mgoDrv.Database {
				return Use(name, AppName(opt.AppName)).Database
			},
			di.Name(name),
		)
	}

	go startDaemonRoutines(ctx, opt.AppName, name, conf)
}

func init() {
	config.AddComponent(config.ComponentMongo, Construct, config.WithFlag(&flagString))
}
