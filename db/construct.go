package db

import (
	"context"
	"log"
	"reflect"
	"sync"
	"syscall"

	"github.com/PaesslerAG/gval"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/infra/drivers/orm"
	"github.com/wfusion/gofusion/common/infra/drivers/orm/idgen"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/db/callbacks"
	"github.com/wfusion/gofusion/db/plugins"
	"github.com/wfusion/gofusion/db/softdelete"
	"github.com/wfusion/gofusion/routine"

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
	// patch delete at
	patches := make([]*gomonkey.Patches, 0, len(confs))
	patches = append(patches, softdelete.PatchGormDeleteAt())

	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if instances != nil {
			for _, instance := range instances[opt.AppName] {
				if sqlDB, err := instance.GetProxy().DB(); err == nil {
					if err := sqlDB.Close(); err != nil {
						log.Printf("%v [Gofusion] %s %s close error: %s", pid, app, config.ComponentDB, err)
					}
				}
			}
			delete(instances, opt.AppName)
		}
		if len(instances) == 0 {
			for _, patch := range patches {
				if patch != nil {
					patch.Reset()
				}
			}
			softdelete.PatchGormDeleteAtOnce = new(sync.Once)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	var logObj logger.Interface
	if !config.Use(opt.AppName).Debug() && utils.IsStrNotBlank(conf.LoggerConfig.Logger) {
		loggerType := inspect.TypeOf(conf.LoggerConfig.Logger)
		loggerValue := reflect.New(loggerType)
		if loggerValue.Type().Implements(customLoggerType) {
			l := fmkLog.Use(conf.LoggerConfig.LogInstance, fmkLog.AppName(opt.AppName))
			loggerValue.Interface().(customLogger).Init(l, opt.AppName, name)
		}
		logObj = loggerValue.Interface().(logger.Interface)
	}

	// conf.Option.Password = config.CryptoDecryptFunc()(conf.Option.Password)
	db, err := orm.Gorm.New(ctx, conf.Option, orm.WithLogger(logObj))
	if err != nil {
		panic(errors.Errorf("initialize gorm db instance error: %+v", err))
	}

	adaptMysqlAutoIncrementIncrement(db, conf)
	mysqlSoftDelete(db, conf)
	if config.Use(opt.AppName).Debug() {
		db.DB = db.Debug()
	}

	// sharding
	tablePluginMap := make(map[string]plugins.TableSharding, len(conf.Sharding))
	for _, shardConf := range conf.Sharding {
		var generator idgen.Generator
		if utils.IsStrNotBlank(shardConf.IDGen) {
			generator = (*(*func() idgen.Generator)(inspect.FuncOf(shardConf.IDGen)))()
		}

		var expression gval.Evaluable
		if utils.IsStrNotBlank(shardConf.ShardingKeyExpr) {
			expression = utils.Must(gval.Full().NewEvaluable(shardConf.ShardingKeyExpr))
		}

		tableShardingPlugin := plugins.DefaultTableSharding(plugins.TableShardingConfig{
			Database:                 name,
			Table:                    shardConf.Table,
			ShardingKeys:             shardConf.Columns,
			ShardingKeyExpr:          expression,
			ShardingKeyByRawValue:    shardConf.ShardingKeyByRawValue,
			ShardingKeysForMigrating: shardConf.ShardingKeysForMigrating,
			NumberOfShards:           shardConf.NumberOfShards,
			CustomSuffix:             shardConf.Suffix,
			PrimaryKeyGenerator:      generator,
		})

		utils.MustSuccess(db.Use(tableShardingPlugin))
		tablePluginMap[shardConf.Table] = tableShardingPlugin
	}

	rwlock.Lock()
	defer rwlock.Unlock()
	if instances == nil {
		instances = make(map[string]map[string]*Instance)
	}
	if instances[opt.AppName] == nil {
		instances[opt.AppName] = make(map[string]*Instance)
	}
	if _, ok := instances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	instances[opt.AppName][name] = &Instance{db: db, name: name, tableShardingPlugins: tablePluginMap}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() *gorm.DB {
				rwlock.RLock()
				defer rwlock.RUnlock()
				return instances[opt.AppName][name].GetProxy()
			},
			di.Name(name),
		)
	}

	routine.Loop(startDaemonRoutines, routine.Args(ctx, opt.AppName, name), routine.AppName(opt.AppName))
}

// adaptAutoIncrementIncrement patch gorm schema parse method to enable changing autoIncrementIncrement in runtime
// see also: https://github.com/go-gorm/gorm/issues/5814
func adaptMysqlAutoIncrementIncrement(db *orm.DB, conf *Conf) {
	autoIncrIncr := conf.AutoIncrementIncrement
	// unset, query auto increment increment
	if autoIncrIncr == 0 && conf.Driver == orm.DriverMysql {
		type autoConfig struct {
			VariableName string `gorm:"column:Variable_name"`
			Value        int64  `gorm:"column:Value"`
		}

		var cfg *autoConfig
		db.WithContext(context.Background()).
			Raw("show variables like 'auto_increment_increment'").
			Scan(&cfg)
		autoIncrIncr = cfg.Value
	}
	// no need to replace callbacks
	if autoIncrIncr <= 1 {
		return
	}

	callbacks.CreateAutoIncr(db.GetProxy(), db.GetDialector(), autoIncrIncr)
}

func mysqlSoftDelete(db *orm.DB, conf *Conf) {
	callbacks.SoftDelete(db.GetProxy())
}

func init() {
	config.AddComponent(config.ComponentDB, Construct)
}
