package softdelete

import (
	"database/sql"
	"database/sql/driver"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type Timestamp sql.NullInt64

// Scan implements the Scanner interface.
func (t *Timestamp) Scan(value any) error {
	return (*sql.NullInt64)(t).Scan(value)
}

// Value implements the driver Valuer interface.
func (t Timestamp) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}
	return t.Int64, nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.Valid {
		return json.Marshal(t.Int64)
	}
	return json.Marshal(nil)
}
func (t *Timestamp) UnmarshalJSON(bs []byte) error {
	if string(bs) == "null" {
		t.Valid = false
		return nil
	}
	err := json.Unmarshal(bs, &t.Int64)
	if err == nil {
		t.Valid = true
	}
	return err
}

func (Timestamp) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{TimestampQueryClause{Field: f, ZeroValue: parseTimestampZeroValueTag(f)}}
}

type TimestampQueryClause struct {
	ZeroValue sql.NullInt64
	Field     *schema.Field
}

func (t TimestampQueryClause) Name() string {
	return ""
}
func (t TimestampQueryClause) Build(clause.Builder) {
}
func (t TimestampQueryClause) MergeClause(*clause.Clause) {
}
func (t TimestampQueryClause) ModifyStatement(stmt *gorm.Statement) {
	if _, ok := stmt.Clauses[timestampEnabledFlag]; ok || stmt.Statement.Unscoped {
		return
	}
	if c, ok := stmt.Clauses["WHERE"]; ok {
		if where, ok := c.Expression.(clause.Where); ok && len(where.Exprs) >= 1 {
			for _, expr := range where.Exprs {
				if orCond, ok := expr.(clause.OrConditions); ok && len(orCond.Exprs) == 1 {
					where.Exprs = []clause.Expression{clause.And(where.Exprs...)}
					c.Expression = where
					stmt.Clauses["WHERE"] = c
					break
				}
			}
		}
	}

	stmt.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: t.Field.DBName}, Value: t.ZeroValue},
	}})
	stmt.Clauses[timestampEnabledFlag] = clause.Clause{}
}

func (Timestamp) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{TimestampUpdateClause{Field: f, ZeroValue: parseTimestampZeroValueTag(f)}}
}

type TimestampUpdateClause struct {
	ZeroValue sql.NullInt64
	Field     *schema.Field
}

func (t TimestampUpdateClause) Name() string {
	return ""
}
func (t TimestampUpdateClause) Build(clause.Builder) {
}
func (t TimestampUpdateClause) MergeClause(*clause.Clause) {
}
func (t TimestampUpdateClause) ModifyStatement(stmt *gorm.Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		TimestampQueryClause(t).ModifyStatement(stmt)
	}
}

func (Timestamp) DeleteClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{TimestampDeleteClause{Field: f, ZeroValue: parseTimestampZeroValueTag(f)}}
}

type TimestampDeleteClause struct {
	ZeroValue sql.NullInt64
	Field     *schema.Field
}

func (t TimestampDeleteClause) Name() string {
	return ""
}
func (t TimestampDeleteClause) Build(clause.Builder) {
}
func (t TimestampDeleteClause) MergeClause(*clause.Clause) {
}
func (t TimestampDeleteClause) ModifyStatement(stmt *gorm.Statement) {
	if stmt.Statement.Unscoped || stmt.SQL.Len() > 0 {
		return
	}

	curTimestamp := utils.GetTimeStamp(stmt.DB.NowFunc())
	setClauses := clause.Set{{Column: clause.Column{Name: t.Field.DBName}, Value: curTimestamp}}
	if clauses, ok := stmt.Clauses[setClauses.Name()]; ok {
		if exprClauses, ok := clauses.Expression.(clause.Set); ok {
			setClauses = append(setClauses, exprClauses...)
		}
	}
	stmt.AddClause(setClauses)
	stmt.SetColumn(t.Field.DBName, curTimestamp, true)

	TimestampQueryClause(t).ModifyStatement(stmt)
}

func parseTimestampZeroValueTag(f *schema.Field) sql.NullInt64 {
	if v, ok := f.TagSettings["ZEROVALUE"]; ok {
		if vv, err := strconv.ParseInt(v, 10, 64); err == nil {
			return sql.NullInt64{Int64: vv, Valid: true}
		}
	}
	return sql.NullInt64{Valid: false}
}
