package callbacks

import (
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"

	"github.com/wfusion/gofusion/db/softdelete"
)

func BuildQuerySQL(db *gorm.DB) {
	callbacks.BuildQuerySQL(db)
}

func BuildCreateSQL(db *gorm.DB) {
	supportReturning := utils.Contains(db.Callback().Create().Clauses, "RETURNING")
	if db.Statement.Schema != nil {
		if !db.Statement.Unscoped {
			for _, c := range db.Statement.Schema.CreateClauses {
				db.Statement.AddClause(c)
			}
		}

		if supportReturning && len(db.Statement.Schema.FieldsWithDefaultDBValue) > 0 {
			if _, ok := db.Statement.Clauses["RETURNING"]; !ok {
				fromColumns := make([]clause.Column, 0, len(db.Statement.Schema.FieldsWithDefaultDBValue))
				for _, field := range db.Statement.Schema.FieldsWithDefaultDBValue {
					fromColumns = append(fromColumns, clause.Column{Name: field.DBName})
				}
				db.Statement.AddClause(clause.Returning{Columns: fromColumns})
			}
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(180)
		db.Statement.AddClauseIfNotExists(clause.Insert{})
		db.Statement.AddClause(callbacks.ConvertToCreateValues(db.Statement))

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func BuildUpdateSQL(db *gorm.DB) {
	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.UpdateClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(180)
		db.Statement.AddClauseIfNotExists(clause.Update{})
		if _, ok := db.Statement.Clauses["SET"]; !ok {
			if set := callbacks.ConvertToAssignments(db.Statement); len(set) != 0 {
				db.Statement.AddClause(set)
			} else {
				return
			}
		}

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func BuildDeleteSQL(db *gorm.DB) {
	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.DeleteClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() > 0 {
		return
	}

	stmt := db.Statement
	stmt.SQL.Grow(180)
	if softdelete.IsClausesWithSoftDelete(db.Statement.Clauses) {
		stmt.AddClauseIfNotExists(clause.Update{})

		if stmt.Schema != nil {
			_, queryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}

			if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema.GetIdentityFieldValuesMap(stmt.Context, reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		}

		stmt.AddClauseIfNotExists(clause.From{})
		stmt.Build(stmt.DB.Callback().Update().Clauses...)
	} else {
		stmt.AddClauseIfNotExists(clause.Delete{})

		if stmt.Schema != nil {
			_, queryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}

			if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema.GetIdentityFieldValuesMap(stmt.Context, reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		}

		stmt.AddClauseIfNotExists(clause.From{})
		stmt.Build(stmt.BuildClauses...)
	}
}
