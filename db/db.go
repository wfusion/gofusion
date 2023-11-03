package db

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/infra/drivers/orm"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/db/plugins"
)

const (
	defaultSlowThreshold = 200 * time.Millisecond
)

var (
	rwlock    = new(sync.RWMutex)
	instances map[string]map[string]*Instance
)

type Instance struct {
	name                 string
	db                   *orm.DB
	tableShardingPlugins map[string]plugins.TableSharding
}

func (d *Instance) GetProxy() *gorm.DB {
	return d.db.GetProxy()
}

type DB struct {
	*orm.DB
	Name                 string
	tableShardingPlugins map[string]plugins.TableSharding
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func Use(ctx context.Context, name string, opts ...utils.OptionExtender) *DB {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := instances[opt.appName]
	if !ok {
		panic(errors.Errorf("db instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("db instance not found for name: %s", name))
	}

	return &DB{DB: instance.db.WithContext(ctx), Name: name, tableShardingPlugins: instance.tableShardingPlugins}
}
