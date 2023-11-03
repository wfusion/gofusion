package callbacks

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"

	comUtl "github.com/wfusion/gofusion/common/utils"

	. "gorm.io/driver/mysql"
)

func CreateAutoIncr(db *gorm.DB, gormDialector gorm.Dialector, autoIncrIncr int64) {
	withReturning := false
	dialector := gormDialector.(*Dialector)
	if !dialector.Config.SkipInitializeWithVersion && strings.Contains(dialector.ServerVersion, "MariaDB") {
		withReturning = checkVersion(dialector.ServerVersion, "10.5")
	}

	lastInsertIDReversed := false
	if !dialector.Config.DisableWithReturning && withReturning {
		lastInsertIDReversed = true
	}

	comUtl.MustSuccess(
		db.Callback().Create().Replace("gorm:create", func(db *gorm.DB) {
			if db.Error != nil {
				return
			}

			BuildCreateSQL(db)

			isDryRun := !db.DryRun && db.Error == nil
			if !isDryRun {
				return
			}

			ok, mode := hasReturning(db, utils.Contains(db.Callback().Create().Clauses, "RETURNING"))
			if ok {
				if c, ok := db.Statement.Clauses["ON CONFLICT"]; ok {
					if onConflict, _ := c.Expression.(clause.OnConflict); onConflict.DoNothing {
						mode |= gorm.ScanOnConflictDoNothing
					}
				}

				rows, err := db.Statement.ConnPool.QueryContext(
					db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...,
				)
				if db.AddError(err) == nil {
					defer func() { _ = db.AddError(rows.Close()) }()
					gorm.Scan(rows, db, mode)
				}

				return
			}

			result, err := db.Statement.ConnPool.ExecContext(
				db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...,
			)
			if err != nil {
				_ = db.AddError(err)
				return
			}

			db.RowsAffected, _ = result.RowsAffected()
			if db.RowsAffected != 0 && db.Statement.Schema != nil &&
				db.Statement.Schema.PrioritizedPrimaryField != nil &&
				db.Statement.Schema.PrioritizedPrimaryField.HasDefaultValue {
				insertID, err := result.LastInsertId()
				insertOk := err == nil && insertID > 0
				if !insertOk {
					_ = db.AddError(err)
					return
				}

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					if lastInsertIDReversed {
						for i := db.Statement.ReflectValue.Len() - 1; i >= 0; i-- {
							rv := db.Statement.ReflectValue.Index(i)
							if reflect.Indirect(rv).Kind() != reflect.Struct {
								break
							}

							_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, rv)
							if isZero {
								_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, rv, insertID))
								//insertID -= db.Statement.Schema.PrioritizedPrimaryField.AutoIncrementIncrement
								insertID -= autoIncrIncr
							}
						}
					} else {
						for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
							rv := db.Statement.ReflectValue.Index(i)
							if reflect.Indirect(rv).Kind() != reflect.Struct {
								break
							}

							if _, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, rv); isZero {
								_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, rv, insertID))
								//insertID += db.Statement.Schema.PrioritizedPrimaryField.AutoIncrementIncrement
								insertID += autoIncrIncr
							}
						}
					}
				case reflect.Struct:
					_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
					if isZero {
						_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, db.Statement.ReflectValue, insertID))
					}
				}
			}
		}),
	)
}
