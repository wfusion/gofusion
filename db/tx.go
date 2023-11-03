package db

import (
	"context"

	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/infra/drivers/orm"
	"github.com/wfusion/gofusion/common/utils"
)

type txOption struct {
	dbName string
}

func TxUse(name string) utils.OptionFunc[txOption] {
	return func(o *txOption) {
		o.dbName = name
	}
}

// WithinTx 事务内执行 DAL 操作
func WithinTx(ctx context.Context, cb func(ctx context.Context) (err error), opts ...utils.OptionExtender) error {
	var db *DB

	o := utils.ApplyOptions[useOption](opts...)
	opt := utils.ApplyOptions[txOption](opts...)
	if opt.dbName == "" {
		db = GetCtxGormDB(ctx)
	} else {
		utils.IfAny(
			func() bool { db = GetCtxGormDBByName(ctx, opt.dbName); return db != nil },
			func() bool { db = Use(ctx, opt.dbName, AppName(o.appName)); return db != nil },
		)
	}
	if db == nil {
		panic(ErrDatabaseNotFound)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return cb(SetCtxGormDB(ctx, &DB{
			DB:                   &orm.DB{DB: tx},
			Name:                 db.Name,
			tableShardingPlugins: db.tableShardingPlugins,
		}))
	})
}
