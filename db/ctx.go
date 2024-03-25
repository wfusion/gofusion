package db

import (
	"context"

	"github.com/wfusion/gofusion/common/utils"

	fusCtx "github.com/wfusion/gofusion/context"
)

func GetCtxGormDB(ctx context.Context) *DB {
	return utils.GetCtxAny(ctx, fusCtx.KeyGormDB, (*DB)(nil))
}

func GetCtxGormDBByName(ctx context.Context, name string) (db *DB) {
	utils.TravelCtx(ctx, func(ctx context.Context) bool {
		db = utils.GetCtxAny(ctx, fusCtx.KeyGormDB, (*DB)(nil))
		return db != nil && db.Name == name
	})
	return
}

func GetCtxGormDBByNameList(ctx context.Context, nameList []string) (db *DB) {
	names := utils.NewSet(nameList...)
	utils.TravelCtx(ctx, func(ctx context.Context) bool {
		db = utils.GetCtxAny(ctx, fusCtx.KeyGormDB, (*DB)(nil))
		return db != nil && names.Contains(db.Name)
	})
	return
}

func SetCtxGormDB(ctx context.Context, db *DB) context.Context {
	return utils.SetCtxAny(ctx, fusCtx.KeyGormDB, db)
}
