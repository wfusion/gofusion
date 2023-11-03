package softdelete

import (
	"database/sql/driver"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type Deleted bool

// Scan implements the Scanner interface.
func (s *Deleted) Scan(value any) (err error) {
	var b bool
	if err = convertAssignRows(&b, value, nil); err != nil {
		return
	}
	*s = Deleted(b)
	return
}

// Value implements the driver Valuer interface.
func (s Deleted) Value() (driver.Value, error) {
	return bool(s), nil
}

func (s Deleted) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(s))
}
func (s *Deleted) UnmarshalJSON(bs []byte) error {
	if string(bs) == "null" {
		return nil
	}
	var b bool
	err := json.Unmarshal(bs, &b)
	if err == nil {
		*s = Deleted(b)
	}

	return err
}

func (Deleted) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{deletedQueryClause{Field: f, ZeroValue: parseStatusZeroValueTag(f)}}
}

type deletedQueryClause struct {
	ZeroValue Deleted
	Field     *schema.Field
}

func (s deletedQueryClause) Name() string {
	return ""
}
func (s deletedQueryClause) Build(clause.Builder) {
}
func (s deletedQueryClause) MergeClause(*clause.Clause) {
}
func (s deletedQueryClause) ModifyStatement(stmt *gorm.Statement) {
	if _, ok := stmt.Clauses[statusEnabledFlag]; ok || stmt.Statement.Unscoped {
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
		clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: s.Field.DBName}, Value: s.ZeroValue},
	}})
	stmt.Clauses[statusEnabledFlag] = clause.Clause{}
}

func (Deleted) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{deletedUpdateClause{Field: f, ZeroValue: parseStatusZeroValueTag(f)}}
}

type deletedUpdateClause struct {
	ZeroValue Deleted
	Field     *schema.Field
}

func (s deletedUpdateClause) Name() string {
	return ""
}
func (s deletedUpdateClause) Build(clause.Builder) {
}
func (s deletedUpdateClause) MergeClause(*clause.Clause) {
}
func (s deletedUpdateClause) ModifyStatement(stmt *gorm.Statement) {
	if stmt.SQL.Len() == 0 && !stmt.Statement.Unscoped {
		deletedQueryClause(s).ModifyStatement(stmt)
	}
}

func (Deleted) DeleteClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{deletedDeleteClause{Field: f, ZeroValue: parseStatusZeroValueTag(f)}}
}

type deletedDeleteClause struct {
	ZeroValue Deleted
	Field     *schema.Field
}

func (s deletedDeleteClause) Name() string {
	return ""
}
func (s deletedDeleteClause) Build(clause.Builder) {
}
func (s deletedDeleteClause) MergeClause(*clause.Clause) {
}
func (s deletedDeleteClause) ModifyStatement(stmt *gorm.Statement) {
	deleted := true
	setClauses := clause.Set{{Column: clause.Column{Name: s.Field.DBName}, Value: deleted}}
	if clauses, ok := stmt.Clauses[setClauses.Name()]; ok {
		if exprClauses, ok := clauses.Expression.(clause.Set); ok {
			setClauses = append(setClauses, exprClauses...)
		}
	}
	stmt.AddClause(setClauses)
	stmt.SetColumn(s.Field.DBName, deleted, true)

	deletedQueryClause(s).ModifyStatement(stmt)
}

func parseStatusZeroValueTag(f *schema.Field) (s Deleted) {
	if v, ok := f.TagSettings["ZEROVALUE"]; ok {
		if vv, err := strconv.ParseBool(v); err == nil {
			return Deleted(vv)
		}
	}
	return Deleted(false)
}
