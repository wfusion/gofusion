package callbacks

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"

	"github.com/wfusion/gofusion/db/softdelete"

	comUtl "github.com/wfusion/gofusion/common/utils"
)

func SoftDelete(db *gorm.DB) {
	// update callback
	comUtl.MustSuccess(db.Callback().Update().Replace("gorm:update", func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		BuildUpdateSQL(db)
		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			if ok, mode := hasReturning(db, utils.Contains(db.Callback().Update().Clauses, "RETURNING")); ok {
				if rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); db.AddError(err) == nil {
					dest := db.Statement.Dest
					db.Statement.Dest = db.Statement.ReflectValue.Addr().Interface()
					gorm.Scan(rows, db, mode)
					db.Statement.Dest = dest
					_ = db.AddError(rows.Close())
				}
			} else {
				result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
				}
			}
		}
	}))

	// delete callback
	comUtl.MustSuccess(db.Callback().Delete().Replace("gorm:delete", func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		BuildDeleteSQL(db)
		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			ok, mode := hasReturning(db, utils.Contains(db.Callback().Delete().Clauses, "RETURNING"))
			if !ok {
				result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
				}

				return
			}

			if rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); db.AddError(err) == nil {
				gorm.Scan(rows, db, mode)
				_ = db.AddError(rows.Close())
			}
		}
	}))

	return
}

func checkMissingWhereConditions(db *gorm.DB) {
	if !db.AllowGlobalUpdate && db.Error == nil {
		where, withCondition := db.Statement.Clauses["WHERE"]
		if withCondition {
			if softdelete.IsClausesWithSoftDelete(db.Statement.Clauses) {
				whereClause, _ := where.Expression.(clause.Where)
				withCondition = len(whereClause.Exprs) > 1
			}
		}
		if !withCondition {
			_ = db.AddError(gorm.ErrMissingWhereClause)
		}
		return
	}
}
