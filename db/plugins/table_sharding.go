package plugins

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/PaesslerAG/gval"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/infra/drivers/orm/idgen"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/sqlparser"
	"github.com/wfusion/gofusion/db/callbacks"
)

const (
	shardingIgnoreStoreKey = "sharding_ignore"
)

var (
	ErrInvalidID             = errors.New("invalid id format")
	ErrIDGeneratorNotFound   = errors.New("id generator not found")
	ErrShardingModelNotFound = errors.New("sharding table model not found when migrating")
	ErrDiffSuffixDML         = errors.New("can not query different suffix table in one sql")
	ErrMissingShardingKey    = errors.New("sharding key required and use operator =")
	ErrColumnAndExprMisMatch = errors.New("column names and expressions mismatch")

	gormSchemaEmbeddedNamer = inspect.TypeOf("gorm.io/gorm/schema.embeddedNamer")
)

type TableShardingConfig struct {
	// Database name
	Database string

	// Table name
	Table string

	// ShardingKeys required, specifies the table columns you want to use for sharding the table rows.
	// For example, for a product order table, you may want to split the rows by `user_id`.
	ShardingKeys []string

	// ShardingKeyExpr optional, specifies how to calculate sharding key by columns, e.g. tenant_id << 16 | user_id
	ShardingKeyExpr gval.Evaluable

	// ShardingKeyByRawValue optional, specifies sharding key with snake values, e.g. xxx_region1_az1, xxx_region1_az2
	ShardingKeyByRawValue bool

	// ShardingKeysForMigrating optional, specifies all sharding keys
	ShardingKeysForMigrating []string

	// NumberOfShards required, specifies how many tables you want to sharding.
	NumberOfShards uint

	// CustomSuffix optional, specifies shard table a custom suffix, e.g. user_%02d means <main_table_name>_user_01
	CustomSuffix string

	// PrimaryKeyGenerator optional, generates id if id is a sharding key and is zero
	PrimaryKeyGenerator idgen.Generator
}

// sharding plugin inspired by gorm.io/sharding@v0.5.3
type tableSharding struct {
	*gorm.DB

	config TableShardingConfig

	shardingFunc              func(ctx context.Context, values ...any) (suffix string, err error)
	isShardingPrimaryKey      bool
	shardingPrimaryKey        string
	shardingTableModel        any
	shardingTableCreatedMutex sync.RWMutex
	shardingTableCreated      map[string]struct{}

	suffixFormat string
}

func DefaultTableSharding(config TableShardingConfig) TableSharding {
	if utils.IsStrBlank(config.Table) {
		panic(errors.New("missing sharding table name"))
	}
	if len(config.ShardingKeys) == 0 {
		panic(errors.New("missing sharding keys"))
	}
	if !config.ShardingKeyByRawValue && (config.NumberOfShards <= 0 || config.NumberOfShards >= 100000) {
		panic(errors.New("invalid number of shards"))
	}

	shardingKeySet := utils.NewSet(config.ShardingKeys...)
	shardingPrimaryKey := ""
	isShardingPrimaryKey := false
	if shardingKeySet.Contains("id") || shardingKeySet.Contains("ID") ||
		shardingKeySet.Contains("iD") || shardingKeySet.Contains("Id") {
		if config.PrimaryKeyGenerator == nil {
			panic(errors.New("sharding by primary key but primary key generator not found"))
		}

		isShardingPrimaryKey = true
		for _, key := range config.ShardingKeys {
			if key == "id" || key == "ID" || key == "Id" || key == "iD" {
				shardingPrimaryKey = key
				break
			}
		}
	}

	return &tableSharding{
		config:               config,
		isShardingPrimaryKey: isShardingPrimaryKey,
		shardingPrimaryKey:   shardingPrimaryKey,
		shardingTableCreated: make(map[string]struct{}, config.NumberOfShards),
	}
}

func (t *tableSharding) Name() string {
	return fmt.Sprintf("gorm:sharding:%s:%s", t.config.Database, t.config.Table)
}

func (t *tableSharding) Initialize(db *gorm.DB) (err error) {
	db.Dialector = newShardingDialector(db.Dialector, t)

	t.DB = db
	t.shardingFunc = t.defaultShardingFunc()
	t.registerCallbacks(db)
	return
}

func (t *tableSharding) ShardingByModelList(ctx context.Context, src ...any) (dst map[string][]any, err error) {
	dst = make(map[string][]any, len(t.config.ShardingKeys))
	for _, m := range src {
		val := reflect.Indirect(reflect.ValueOf(m))
		shardingValues := make([]any, 0, len(t.config.ShardingKeys))
		for _, key := range t.config.ShardingKeys {
			field := val.FieldByNameFunc(func(v string) bool { return strings.EqualFold(v, key) })
			if !field.IsValid() {
				field, _ = utils.GetGormColumnValue(val, key)
			}
			if !field.IsValid() {
				return dst, ErrMissingShardingKey
			}
			if key == t.shardingPrimaryKey && field.IsZero() {
				return dst, ErrInvalidID
			}
			shardingValues = append(shardingValues, field.Interface())
		}
		suffix, err := t.shardingFunc(ctx, shardingValues...)
		if err != nil {
			return dst, err
		}
		dst[suffix] = append(dst[suffix], m)
	}
	return
}

func (t *tableSharding) ShardingByValues(ctx context.Context, src []map[string]any) (
	dst map[string][]map[string]any, err error) {
	dst = make(map[string][]map[string]any, len(t.config.ShardingKeys))
	for _, col := range src {
		values := make([]any, 0, len(col))
		for _, k := range t.config.ShardingKeys {
			value, ok := col[k]
			if !ok {
				return dst, errors.Errorf("sharding key not found [column[%s]]", k)
			}
			if k == t.shardingPrimaryKey && utils.IsBlank(value) {
				return dst, ErrInvalidID
			}
			values = append(values, value)
		}
		suffix, err := t.shardingFunc(ctx, values...)
		if err != nil {
			return dst, err
		}
		dst[suffix] = append(dst[suffix], col)
	}
	return
}

func (t *tableSharding) ShardingIDGen(ctx context.Context) (id uint64, err error) {
	if t.config.PrimaryKeyGenerator == nil {
		return 0, ErrIDGeneratorNotFound
	}
	return t.config.PrimaryKeyGenerator.Next()
}

func (t *tableSharding) registerCallbacks(db *gorm.DB) {
	utils.MustSuccess(db.Callback().
		Create().
		After("gorm:before_create").
		Before("gorm:save_before_associations").
		Register(t.Name(), t.createCallback))

	utils.MustSuccess(db.Callback().
		Query().
		Before("gorm:query").
		Register(t.Name(), t.queryCallback))

	utils.MustSuccess(db.Callback().
		Update().
		After("gorm:before_update").
		Before("gorm:save_before_associations").
		Register(t.Name(), t.updateCallback))

	utils.MustSuccess(db.Callback().
		Delete().
		After("gorm:before_delete").
		Before("gorm:delete_before_associations").
		Register(t.Name(), t.deleteCallback))

	utils.MustSuccess(db.Callback().
		Row().
		Before("gorm:row").
		Register(t.Name(), t.queryCallback))

	utils.MustSuccess(db.Callback().
		Raw().
		Before("gorm:raw").
		Register(t.Name(), t.rawCallback))
}
func (t *tableSharding) createCallback(db *gorm.DB) {
	utils.IfAny(
		t.isIgnored(db),
		func() bool { ok1, ok2 := t.dispatchTableByModel(db, tableShardingIsInsert()); return ok1 || ok2 },
		func() bool {
			callbacks.BuildCreateSQL(db)
			t.wrapDispatchTableBySQL(db, tableShardingIsInsert())
			return true
		},
	)
}
func (t *tableSharding) queryCallback(db *gorm.DB) {
	utils.IfAny(
		t.isIgnored(db),
		func() bool { ok1, ok2 := t.dispatchTableByModel(db); return ok1 || ok2 },
		func() bool {
			callbacks.BuildQuerySQL(db)
			t.wrapDispatchTableBySQL(db)
			return true
		},
	)
}
func (t *tableSharding) updateCallback(db *gorm.DB) {
	utils.IfAny(
		t.isIgnored(db),
		func() bool { ok1, ok2 := t.dispatchTableByModel(db); return ok1 || ok2 },
		func() bool {
			callbacks.BuildUpdateSQL(db)
			t.wrapDispatchTableBySQL(db)
			return true
		},
	)
}
func (t *tableSharding) deleteCallback(db *gorm.DB) {
	utils.IfAny(
		t.isIgnored(db),
		func() bool { ok1, ok2 := t.dispatchTableByModel(db); return ok1 || ok2 },
		func() bool {
			callbacks.BuildDeleteSQL(db)
			t.wrapDispatchTableBySQL(db)
			return true
		},
	)
}
func (t *tableSharding) rawCallback(db *gorm.DB) {
	utils.IfAny(
		t.isIgnored(db),
		func() bool { ok1, ok2 := t.dispatchTableByModel(db); return ok1 || ok2 },
		func() bool { t.wrapDispatchTableBySQL(db); return true },
	)
}

type tableShardingDispatchOption struct {
	isInsert bool
}

func tableShardingIsInsert() utils.OptionFunc[tableShardingDispatchOption] {
	return func(t *tableShardingDispatchOption) {
		t.isInsert = true
	}
}

func (t *tableSharding) dispatchTableByModel(db *gorm.DB, opts ...utils.OptionExtender) (otherTable, ok bool) {
	if db.Statement.Model == nil || utils.IsBlank(db.Statement.ReflectValue.Interface()) {
		return
	}
	if db.Statement.Table != t.config.Table {
		otherTable = true
		return
	}
	if t.shardingTableModel == nil {
		if _, ok := db.Statement.Model.(schema.Tabler); ok {
			cloneModel := clone.Clone(db.Statement.Model)
			t.shardingTableModel = cloneModel
		}
	}

	opt := utils.ApplyOptions[tableShardingDispatchOption](opts...)
	if t.isShardingPrimaryKey {
		if err := t.setPrimaryKeyByModel(db, opt); err != nil {
			_ = db.AddError(err)
			return
		}
	}

	reflectVal, ok := t.getModelReflectValue(db)
	if !ok {
		return
	}
	if err := t.checkDiffSuffixesByModel(db); err != nil {
		return
	}

	values := make([]any, 0, len(t.config.ShardingKeys))
	for _, key := range t.config.ShardingKeys {
		val := reflectVal.FieldByNameFunc(func(v string) bool { return strings.EqualFold(v, key) })
		if !val.IsValid() {
			val, _ = utils.GetGormColumnValue(reflectVal, key)
		}
		if !val.IsValid() {
			_ = db.AddError(ErrMissingShardingKey)
			return
		}
		values = append(values, val.Interface())
	}

	suffix, err := t.shardingFunc(db.Statement.Context, values...)
	if err != nil {
		_ = db.AddError(err)
		return
	}
	// cannot parse suffix from model
	if utils.IsStrBlank(suffix) || suffix == constant.Underline {
		return false, false
	}
	if err = t.createTableIfNotExists(db, db.Statement.Table, suffix); err != nil {
		_ = db.AddError(err)
		return
	}

	db.Statement.Table = db.Statement.Table + suffix
	t.replaceStatementClauseAndSchema(db, opt)
	ok = true
	return
}

//nolint: revive // sql parser issue
func (t *tableSharding) dispatchTableBySQL(db *gorm.DB, opts ...utils.OptionExtender) (ok bool, err error) {
	expr, err := sqlparser.NewParser(strings.NewReader(db.Statement.SQL.String())).ParseStatement()
	if err != nil {
		// maybe not a dml, so we ignore this error
		return
	}

	getSuffix := func(condition sqlparser.Node, tableName string, vars ...any) (suffix string, err error) {
		values := make([]any, 0, len(t.config.ShardingKeys))
		for _, key := range t.config.ShardingKeys {
			val, err := t.nonInsertValue(condition, key, tableName, vars...)
			if err != nil {
				return "", db.AddError(err)
			}
			values = append(values, val)
		}

		suffix, err = t.shardingFunc(db.Statement.Context, values...)
		if err != nil {
			return "", db.AddError(err)
		}
		return
	}

	newSQL := ""
	switch stmt := expr.(type) {
	case *sqlparser.InsertStatement:
		if stmt.TableName.TableName() != t.config.Table {
			return
		}

		suffix := ""
		for _, insertExpression := range stmt.Expressions {
			values, id, e := t.insertValue(t.config.ShardingKeys, stmt.ColumnNames,
				insertExpression.Exprs, db.Statement.Vars...)
			if e != nil {
				_ = db.AddError(e)
				return
			}
			if t.isShardingPrimaryKey && id == 0 {
				if t.config.PrimaryKeyGenerator == nil {
					_ = db.AddError(ErrIDGeneratorNotFound)
					return
				}
				if id, e = t.config.PrimaryKeyGenerator.Next(idgen.GormTx(db)); e != nil {
					_ = db.AddError(e)
					return
				}
				stmt.ColumnNames = append(stmt.ColumnNames, &sqlparser.Ident{Name: "id"})
				insertExpression.Exprs = append(insertExpression.Exprs, &sqlparser.NumberLit{Value: cast.ToString(id)})
				values, _, _ = t.insertValue(t.config.ShardingKeys, stmt.ColumnNames,
					insertExpression.Exprs, db.Statement.Vars...)
			}

			subSuffix, e := t.shardingFunc(db.Statement.Context, values...)
			if e != nil {
				_ = db.AddError(e)
				return
			}

			if suffix != "" && suffix != subSuffix {
				_ = db.AddError(ErrDiffSuffixDML)
				return
			}
			suffix = subSuffix
		}
		// FIXME: could not find the table schema to migrate
		if e := t.createTableIfNotExists(db, db.Statement.Table, suffix); e != nil {
			_ = db.AddError(e)
			return
		}
		stmt.TableName = &sqlparser.TableName{Name: &sqlparser.Ident{Name: stmt.TableName.TableName() + suffix}}
		newSQL = stmt.String()
	case *sqlparser.SelectStatement:
		parseSelectStatementFunc := func(stmt *sqlparser.SelectStatement) (ok bool, err error) {
			if stmt.Hint != nil && stmt.Hint.Value == "nosharding" {
				return false, nil
			}

			switch tbl := stmt.FromItems.(type) {
			case *sqlparser.TableName:
				if tbl.TableName() != t.config.Table {
					return false, nil
				}
				suffix, e := getSuffix(stmt.Condition, t.config.Table, db.Statement.Vars...)
				if e != nil {
					_ = db.AddError(e)
					return false, nil
				}
				oldTableName := tbl.TableName()
				newTableName := oldTableName + suffix
				stmt.FromItems = &sqlparser.TableName{Name: &sqlparser.Ident{Name: newTableName}}
				stmt.OrderBy = t.replaceOrderByTableName(stmt.OrderBy, oldTableName, newTableName)
				if e := t.replaceCondition(stmt.Condition, oldTableName, newTableName); err != nil {
					_ = db.AddError(e)
					return false, nil
				}
			case *sqlparser.JoinClause:
				tblx, _ := tbl.X.(*sqlparser.TableName)
				tbly, _ := tbl.Y.(*sqlparser.TableName)
				isXSharding := tblx != nil && tblx.TableName() == t.config.Table
				isYSharding := tbly != nil && tbly.TableName() == t.config.Table
				oldTableName := ""
				switch {
				case isXSharding:
					oldTableName = tblx.TableName()
				case isYSharding:
					oldTableName = tbly.TableName()
				default:
					return false, nil
				}
				suffix, e := getSuffix(stmt.Condition, oldTableName, db.Statement.Vars...)
				if e != nil {
					_ = db.AddError(e)
					return false, nil
				}
				newTableName := oldTableName + suffix
				stmt.OrderBy = t.replaceOrderByTableName(stmt.OrderBy, oldTableName, newTableName)
				if e := t.replaceCondition(stmt.Condition, oldTableName, newTableName); err != nil {
					_ = db.AddError(e)
					return false, nil
				}
				if e := t.replaceConstraint(tbl.Constraint, oldTableName, newTableName); err != nil {
					_ = db.AddError(e)
					return false, nil
				}
				if isXSharding {
					tblx.Name.Name = newTableName
				} else {
					tbly.Name.Name = newTableName
				}
				if stmt.Columns != nil {
					for _, column := range *stmt.Columns {
						columnTbl, ok := column.Expr.(*sqlparser.QualifiedRef)
						if !ok || columnTbl.Table.Name != oldTableName {
							continue
						}
						columnTbl.Table.Name = newTableName
					}
				}
			}
			return true, nil
		}
		for compound := stmt; compound != nil; compound = compound.Compound {
			if ok, err = parseSelectStatementFunc(compound); !ok || err != nil {
				return
			}
		}

		newSQL = stmt.String()

	case *sqlparser.UpdateStatement:
		if stmt.TableName.TableName() != t.config.Table {
			return
		}

		suffix, e := getSuffix(stmt.Condition, t.config.Table, db.Statement.Vars...)
		if e != nil {
			_ = db.AddError(e)
			return
		}

		oldTableName := stmt.TableName.TableName()
		newTableName := oldTableName + suffix
		stmt.TableName = &sqlparser.TableName{Name: &sqlparser.Ident{Name: newTableName}}
		if e := t.replaceCondition(stmt.Condition, oldTableName, newTableName); err != nil {
			_ = db.AddError(e)
			return false, nil
		}
		newSQL = stmt.String()
	case *sqlparser.DeleteStatement:
		if stmt.TableName.TableName() != t.config.Table {
			return
		}

		suffix, e := getSuffix(stmt.Condition, t.config.Table, db.Statement.Vars...)
		if e != nil {
			_ = db.AddError(e)
			return
		}

		oldTableName := stmt.TableName.TableName()
		newTableName := oldTableName + suffix
		stmt.TableName = &sqlparser.TableName{Name: &sqlparser.Ident{Name: newTableName}}
		if e := t.replaceCondition(stmt.Condition, oldTableName, newTableName); err != nil {
			_ = db.AddError(e)
			return false, nil
		}
		newSQL = stmt.String()
	default:
		_ = db.AddError(sqlparser.ErrNotImplemented)
		return
	}

	sb := strings.Builder{}
	sb.Grow(len(newSQL))
	sb.WriteString(newSQL)
	db.Statement.SQL = sb

	return true, nil
}
func (t *tableSharding) wrapDispatchTableBySQL(db *gorm.DB, opts ...utils.OptionExtender) {
	if ok, err := t.dispatchTableBySQL(db, opts...); err != nil || !ok {
		// not a dml
		if err != nil {
			return
		}
		// not a sharding table
		if !ok {
			// FIXME: reset sql parse result will get duplicated sql statement
			// db.Statement.SQL = strings.Builder{}
			// db.Statement.Vars = nil
		}
	}
}
func (t *tableSharding) replaceStatementClauseAndSchema(db *gorm.DB, opt *tableShardingDispatchOption) {
	changeExprFunc := func(src []clause.Expression) (dst []clause.Expression) {
		changeTableFunc := func(src any) (dst any, ok bool) {
			switch col := src.(type) {
			case clause.Column:
				if col.Table == t.config.Table {
					col.Table = db.Statement.Table
					return col, true
				}
			case clause.Table:
				if col.Name == t.config.Table {
					col.Name = db.Statement.Table
					return col, true
				}
			}
			return
		}
		dst = make([]clause.Expression, 0, len(src))
		for _, srcExpr := range src {
			switch expr := srcExpr.(type) {
			case clause.IN:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Eq:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Neq:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Gt:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Gte:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Lt:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Lte:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			case clause.Like:
				if col, ok := changeTableFunc(expr.Column); ok {
					expr.Column = col
				}
				dst = append(dst, expr)
			default:
				dst = append(dst, expr)
			}
		}
		return
	}
	changeClausesMapping := map[string]func(cls clause.Clause){
		"WHERE": func(cls clause.Clause) {
			whereClause, ok := cls.Expression.(clause.Where)
			if !ok {
				return
			}
			whereClause.Exprs = changeExprFunc(whereClause.Exprs)
			cls.Expression = whereClause
			db.Statement.Clauses["WHERE"] = cls
		},
		"FROM": func(cls clause.Clause) {
			fromClause, ok := cls.Expression.(clause.From)
			if !ok {
				return
			}
			tables := make([]clause.Table, 0, len(fromClause.Tables))
			for _, table := range fromClause.Tables {
				if table.Name == t.config.Table {
					table.Name = db.Statement.Table
					tables = append(tables, table)
				} else {
					tables = append(tables, table)
				}
			}
			fromClause.Tables = tables
			cls.Expression = fromClause
			db.Statement.Clauses["FROM"] = cls
		},
		// TODO: check if order by contains table name
		"ORDER BY": func(cls clause.Clause) {
			_, ok := cls.Expression.(clause.OrderBy)
			if !ok {
				return
			}
		},
	}

	for name, cls := range db.Statement.Clauses {
		if mappingFunc, ok := changeClausesMapping[name]; ok {
			mappingFunc(cls)
		}
	}

	if opt.isInsert {
		db.Clauses(clause.Insert{Table: clause.Table{Name: db.Statement.Table}})
	} else {
		db.Clauses(clause.From{Tables: []clause.Table{{Name: db.Statement.Table}}})
	}
}

func (t *tableSharding) replaceCondition(conditions sqlparser.Expr, oldTableName, newTableName string) (err error) {
	err = sqlparser.Walk(
		sqlparser.VisitFunc(func(node sqlparser.Node) (err error) {
			n, ok := node.(*sqlparser.BinaryExpr)
			if !ok {
				return
			}

			x, ok := n.X.(*sqlparser.QualifiedRef)
			if !ok || x.Table == nil || x.Table.Name != oldTableName {
				return
			}

			x.Table.Name = newTableName
			return
		}),
		conditions,
	)
	return
}

func (t *tableSharding) replaceConstraint(constraints sqlparser.Node, oldTableName, newTableName string) (err error) {
	return sqlparser.Walk(
		sqlparser.VisitFunc(func(node sqlparser.Node) (err error) {
			n, ok := node.(*sqlparser.QualifiedRef)
			if !ok || n.Table == nil || n.Table.Name != oldTableName {
				return
			}

			n.Table.Name = newTableName
			return
		}),
		constraints,
	)
}

func (t *tableSharding) insertValue(keys []string, names []*sqlparser.Ident, exprs []sqlparser.Expr, args ...any) (
	values []any, id uint64, err error) {
	if len(names) != len(exprs) {
		return nil, 0, ErrColumnAndExprMisMatch
	}

	for _, key := range keys {
		found := false
		isPrimaryKey := key == t.shardingPrimaryKey
		for i, name := range names {
			if name.Name != key {
				continue
			}

			switch expr := exprs[i].(type) {
			case *sqlparser.BindExpr:
				if !isPrimaryKey {
					values = append(values, args[expr.Pos])
				} else {
					switch v := args[expr.Pos].(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
						if id, err = cast.ToUint64E(v); err != nil {
							return nil, 0, errors.Wrapf(err, "parse id as uint64 failed [%v]", v)
						}
					default:
						return nil, 0, ErrInvalidID
					}
					if id != 0 {
						values = append(values, args[expr.Pos])
					}
				}
			case *sqlparser.StringLit:
				if !isPrimaryKey {
					values = append(values, expr.Value)
				} else {
					if id, err = cast.ToUint64E(expr.Value); err != nil {
						return nil, 0, errors.Wrapf(err, "parse id as uint64 failed [%s]", expr.Value)
					}
					if id != 0 {
						values = append(values, expr.Value)
					}
				}
			case *sqlparser.NumberLit:
				if !isPrimaryKey {
					values = append(values, expr.Value)
				} else {
					if id, err = strconv.ParseUint(expr.Value, 10, 64); err != nil {
						return nil, 0, errors.Wrapf(err,
							"parse id as uint64 failed [%s]", expr.Value)
					}
					if id != 0 {
						values = append(values, expr.Value)
					}
				}
			default:
				return nil, 0, sqlparser.ErrNotImplemented
			}

			found = true
			break
		}
		if !found && !isPrimaryKey {
			return nil, 0, ErrMissingShardingKey
		}
	}

	return
}

func (t *tableSharding) nonInsertValue(condition sqlparser.Node, key, tableName string, args ...any) (
	value any, err error) {
	found := false
	err = sqlparser.Walk(
		sqlparser.VisitFunc(func(node sqlparser.Node) (err error) {
			n, ok := node.(*sqlparser.BinaryExpr)
			if !ok {
				return
			}
			if n.Op != sqlparser.EQ {
				return
			}

			switch x := n.X.(type) {
			case *sqlparser.Ident:
				if x.Name != key {
					return
				}
			case *sqlparser.QualifiedRef:
				if !ok || x.Table.Name != tableName || x.Column.Name != key {
					return
				}
			}

			found = true
			switch expr := n.Y.(type) {
			case *sqlparser.BindExpr:
				value = args[expr.Pos]
			case *sqlparser.StringLit:
				value = expr.Value
			case *sqlparser.NumberLit:
				value = expr.Value
			default:
				return sqlparser.ErrNotImplemented
			}

			return
		}),
		condition,
	)
	if err != nil {
		return
	}
	if !found {
		return nil, ErrMissingShardingKey
	}
	return
}

func (t *tableSharding) setPrimaryKeyByModel(db *gorm.DB, opt *tableShardingDispatchOption) (err error) {
	if !opt.isInsert || db.Statement.Model == nil ||
		db.Statement.Schema == nil || db.Statement.Schema.PrioritizedPrimaryField == nil {
		return
	}
	setPrimaryKeyFunc := func(rv reflect.Value) (err error) {
		_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, rv)
		if !isZero {
			return
		}
		if t.config.PrimaryKeyGenerator == nil {
			return ErrIDGeneratorNotFound
		}
		id, err := t.config.PrimaryKeyGenerator.Next(idgen.GormTx(db))
		if err != nil {
			return
		}
		return db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, rv, id)
	}

	switch db.Statement.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
			rv := db.Statement.ReflectValue.Index(i)
			if reflect.Indirect(rv).Kind() != reflect.Struct {
				break
			}

			if err = setPrimaryKeyFunc(rv); err != nil {
				return
			}
		}
	case reflect.Struct:
		if err = setPrimaryKeyFunc(db.Statement.ReflectValue); err != nil {
			return
		}
	}

	return
}

func (t *tableSharding) getModelReflectValue(db *gorm.DB) (reflectVal reflect.Value, ok bool) {
	reflectVal = utils.IndirectValue(db.Statement.ReflectValue)
	if reflectVal.Kind() == reflect.Array || reflectVal.Kind() == reflect.Slice {
		if reflectVal.Len() == 0 {
			return
		}
		reflectVal = utils.IndirectValue(reflectVal.Index(0))
	}

	if reflectVal.Kind() != reflect.Struct {
		return
	}

	return reflectVal, !utils.IsBlank(reflectVal.Interface())
}

func (t *tableSharding) checkDiffSuffixesByModel(db *gorm.DB) (err error) {
	reflectVal := utils.IndirectValue(db.Statement.ReflectValue)
	if reflectVal.Kind() != reflect.Array && reflectVal.Kind() != reflect.Slice {
		return
	}

	suffix := ""
	for i := 0; i < reflectVal.Len(); i++ {
		reflectItemVal := reflect.Indirect(reflectVal.Index(i))
		values := make([]any, 0, len(t.config.ShardingKeys))
		for _, key := range t.config.ShardingKeys {
			val := reflectItemVal.FieldByNameFunc(func(v string) bool { return strings.EqualFold(v, key) })
			if !val.IsValid() {
				val, _ = utils.GetGormColumnValue(reflectItemVal, key)
			}
			if !val.IsValid() {
				return db.AddError(ErrMissingShardingKey)
			}
			values = append(values, val.Interface())
		}
		subSuffix, err := t.shardingFunc(db.Statement.Context, values...)
		if err != nil {
			return db.AddError(err)
		}
		if suffix != "" && suffix != subSuffix {
			return db.AddError(ErrDiffSuffixDML)
		}
		suffix = subSuffix
	}
	return
}

func (t *tableSharding) replaceOrderByTableName(
	orderBy []*sqlparser.OrderingTerm, oldName, newName string) []*sqlparser.OrderingTerm {
	for i, term := range orderBy {
		if x, ok := term.X.(*sqlparser.QualifiedRef); ok {
			if x.Table.Name == oldName {
				x.Table.Name = newName
				orderBy[i].X = x
			}
		}
	}
	return orderBy
}

func (t *tableSharding) createTableIfNotExists(db *gorm.DB, tableName, suffix string) (err error) {
	shardingTableName := tableName + suffix
	t.shardingTableCreatedMutex.RLock()
	if _, ok := t.shardingTableCreated[shardingTableName]; ok {
		t.shardingTableCreatedMutex.RUnlock()
		return
	}
	t.shardingTableCreatedMutex.RUnlock()
	t.shardingTableCreatedMutex.Lock()
	defer t.shardingTableCreatedMutex.Unlock()

	defer t.ignore(t.DB)() //nolint: revive // partial calling issue
	if t.DB.Migrator().HasTable(shardingTableName) {
		t.shardingTableCreated[shardingTableName] = struct{}{}
		return
	}

	model := db.Statement.Model
	if model == nil {
		model = t.shardingTableModel
	}
	if model == nil {
		return ErrShardingModelNotFound
	}
	tx := t.DB.Session(&gorm.Session{}).Table(shardingTableName)
	if err = db.Dialector.Migrator(tx).AutoMigrate(db.Statement.Model); err != nil {
		return err
	}
	t.shardingTableCreated[shardingTableName] = struct{}{}
	return
}

func (t *tableSharding) suffixes() (suffixes []string, err error) {
	switch {
	case t.config.ShardingKeyByRawValue:
		if len(t.config.ShardingKeysForMigrating) == 0 {
			return nil, errors.New("sharding key by raw value but do not configure keys for migrating")
		}

		for _, shardingKey := range t.config.ShardingKeysForMigrating {
			suffixes = append(suffixes, fmt.Sprintf(t.suffixFormat, shardingKey))
		}
	default:
		for i := 0; i < int(t.config.NumberOfShards); i++ {
			suffixes = append(suffixes, fmt.Sprintf(t.suffixFormat, i))
		}
	}
	return
}

func (t *tableSharding) ignore(db *gorm.DB) func() {
	if _, ok := db.Statement.Settings.Load(shardingIgnoreStoreKey); ok {
		return func() {}
	}
	db.Statement.Settings.Store(shardingIgnoreStoreKey, nil)
	return func() {
		db.Statement.Settings.Delete(shardingIgnoreStoreKey)
	}
}
func (t *tableSharding) isIgnored(db *gorm.DB) func() bool {
	return func() bool {
		_, ok := db.Statement.Settings.Load(shardingIgnoreStoreKey)
		return ok
	}
}

func (t *tableSharding) defaultShardingFunc() func(ctx context.Context, values ...any) (suffix string, err error) {
	if !t.config.ShardingKeyByRawValue && t.config.NumberOfShards == 0 {
		panic(errors.New("missing number_of_shards config"))
	}
	t.suffixFormat = constant.Underline

	switch {
	case utils.IsStrNotBlank(t.config.CustomSuffix):
		t.suffixFormat += t.config.CustomSuffix
	case t.config.ShardingKeyByRawValue:
		t.suffixFormat += "%s"
	default:
		t.suffixFormat += strings.Join(t.config.ShardingKeys, constant.Underline)
	}

	numberOfShards := t.config.NumberOfShards
	if !strings.Contains(t.suffixFormat, "%") {
		if t.config.ShardingKeyByRawValue {
			t.suffixFormat += "_%s"
		} else if numberOfShards < 10 {
			t.suffixFormat += "_%01d"
		} else if numberOfShards < 100 {
			t.suffixFormat += "_%02d"
		} else if numberOfShards < 1000 {
			t.suffixFormat += "_%03d"
		} else if numberOfShards < 10000 {
			t.suffixFormat += "_%04d"
		}
	}

	switch {
	case t.config.ShardingKeyByRawValue:
		return func(ctx context.Context, values ...any) (suffix string, err error) {
			data := make([]string, 0, len(values))
			for _, value := range values {
				v, err := cast.ToStringE(value)
				if err != nil {
					return "", err
				}
				data = append(data, v)
			}
			shardingKey := strings.Join(data, constant.Underline)
			return fmt.Sprintf("_%s", shardingKey), nil
		}
	case t.config.ShardingKeyExpr != nil:
		numberOfShardsFloat64 := float64(numberOfShards)
		return func(ctx context.Context, values ...any) (suffix string, err error) {
			params := make(map[string]any, len(t.config.ShardingKeys))
			for idx, column := range t.config.ShardingKeys {
				params[column] = values[idx]
			}

			result, err := t.config.ShardingKeyExpr(ctx, params)
			if err != nil {
				return
			}
			shardingKey := int64(math.Mod(cast.ToFloat64(result), numberOfShardsFloat64))
			return fmt.Sprintf(t.suffixFormat, shardingKey), nil
		}
	default:
		stringToByteSliceFunc := func(v string) (data []byte) {
			utils.IfAny(
				// number
				func() (ok bool) {
					num := new(big.Float)
					if _, ok = num.SetString(v); !ok {
						return
					}
					gobEncoded, err := num.GobEncode()
					if err != nil {
						return false
					}
					data = gobEncoded
					return
				},
				// uuid
				func() bool {
					uid, err := uuid.Parse(v)
					if err != nil {
						return false
					}
					data = uid[:]
					return true
				},
				// bytes
				func() bool { data = []byte(v); return true },
			)
			return
		}
		return func(ctx context.Context, values ...any) (suffix string, err error) {
			size := 0
			for _, value := range values {
				s := binary.Size(value)
				if s <= 0 {
					s = int(unsafe.Sizeof(value))
				}
				size += s
			}
			w := new(bytes.Buffer)
			w.Grow(size)

			for _, value := range values {
				var data any
				switch v := value.(type) {
				case int, *int:
					data = utils.IntNarrow(cast.ToInt(v))
				case uint, *uint:
					data = utils.UintNarrow(cast.ToUint(v))
				case []int:
					data = make([]any, len(v))
					for i := 0; i < len(v); i++ {
						data.([]any)[i] = utils.IntNarrow(cast.ToInt(v))
					}
				case []uint:
					data = make([]any, len(v))
					for i := 0; i < len(v); i++ {
						data.([]any)[i] = utils.UintNarrow(cast.ToUint(v))
					}
				case string:
					data = stringToByteSliceFunc(v)
				case []byte:
					data = stringToByteSliceFunc(utils.UnsafeBytesToString(v))
				case uuid.UUID:
					data = v[:]
				default:
					data = v
				}
				if err = binary.Write(w, binary.BigEndian, data); err != nil {
					return
				}
			}

			// checksum mod shards
			checksum := crc32.ChecksumIEEE(w.Bytes())
			shardingKey := uint64(checksum) % uint64(numberOfShards)
			suffix = fmt.Sprintf(t.suffixFormat, shardingKey)
			return
		}
	}
}

type shardingDialector struct {
	gorm.Dialector
	shardingMap map[string]*tableSharding
}

func newShardingDialector(d gorm.Dialector, s *tableSharding) shardingDialector {
	if sd, ok := d.(shardingDialector); ok {
		sd.shardingMap[s.config.Table] = s
		return sd
	}

	return shardingDialector{
		Dialector:   d,
		shardingMap: map[string]*tableSharding{s.config.Table: s},
	}
}

func (s shardingDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := s.Dialector.Migrator(db)
	if (*tableSharding)(nil).isIgnored(db)() {
		return m
	}
	return &shardingMigrator{
		Migrator:    m,
		db:          db,
		shardingMap: s.shardingMap,
		dialector:   s.Dialector,
	}
}
func (s shardingDialector) SavePoint(tx *gorm.DB, name string) error {
	if savePointer, ok := s.Dialector.(gorm.SavePointerDialectorInterface); ok {
		return savePointer.SavePoint(tx, name)
	} else {
		return gorm.ErrUnsupportedDriver
	}
}
func (s shardingDialector) RollbackTo(tx *gorm.DB, name string) error {
	if savePointer, ok := s.Dialector.(gorm.SavePointerDialectorInterface); ok {
		return savePointer.RollbackTo(tx, name)
	} else {
		return gorm.ErrUnsupportedDriver
	}
}

type shardingMigrator struct {
	gorm.Migrator
	db          *gorm.DB
	dialector   gorm.Dialector
	shardingMap map[string]*tableSharding
}

func (s *shardingMigrator) AutoMigrate(dst ...any) (err error) {
	sharding, ok := s.shardingMap[s.tableName(s.db, dst[0])]
	if !ok {
		defer (*tableSharding)(nil).ignore(s.db)() //nolint: revive // partial calling issue
		return s.Migrator.AutoMigrate(dst...)
	}

	stmt := &gorm.Statement{DB: sharding.DB}
	if sharding.isIgnored(sharding.DB)() {
		return s.dialector.Migrator(stmt.DB.Session(&gorm.Session{})).AutoMigrate(dst...)
	}

	shardingDst, err := s.getShardingDst(sharding, dst...)
	if err != nil {
		return err
	}

	defer sharding.ignore(sharding.DB)() //nolint: revive // partial calling issue
	for _, sd := range shardingDst {
		tx := stmt.DB.Session(&gorm.Session{}).Table(sd.table)
		if err = s.dialector.Migrator(tx).AutoMigrate(sd.dst); err != nil {
			return err
		}
	}

	return
}
func (s *shardingMigrator) DropTable(dst ...any) (err error) {
	sharding, ok := s.shardingMap[s.tableName(s.db, dst[0])]
	if !ok {
		defer (*tableSharding)(nil).ignore(s.db)() //nolint: revive // partial calling issue
		return s.Migrator.DropTable(dst...)
	}

	stmt := &gorm.Statement{DB: sharding.DB}
	if sharding.isIgnored(sharding.DB)() {
		return s.dialector.Migrator(stmt.DB.Session(&gorm.Session{})).DropTable(dst...)
	}
	shardingDst, err := s.getShardingDst(sharding, dst...)
	if err != nil {
		return err
	}

	defer sharding.ignore(sharding.DB)() //nolint: revive // partial calling issue
	for _, sd := range shardingDst {
		tx := stmt.DB.Session(&gorm.Session{}).Table(sd.table)
		if err = s.dialector.Migrator(tx).DropTable(sd.table); err != nil {
			return err
		}
	}

	return
}

type shardingDst struct {
	table string
	dst   any
}

func (s *shardingMigrator) getShardingDst(sharding *tableSharding, src ...any) (dst []shardingDst, err error) {
	for _, model := range src {
		stmt := &gorm.Statement{DB: sharding.DB}
		if err = stmt.Parse(model); err != nil {
			return
		}

		// support sharding table
		suffixes, err := sharding.suffixes()
		if err != nil {
			return nil, err
		}
		if len(suffixes) == 0 {
			return nil, fmt.Errorf("sharding table:%s suffixes are empty", stmt.Table)
		}
		for _, suffix := range suffixes {
			dst = append(dst, shardingDst{
				table: stmt.Table + suffix,
				dst:   model,
			})
		}
	}
	return
}
func (s *shardingMigrator) tableName(db *gorm.DB, m any) (name string) {
	if tabler, ok := m.(schema.Tabler); ok {
		name = tabler.TableName()
	}
	if tabler, ok := m.(schema.TablerWithNamer); ok {
		name = tabler.TableName(db.NamingStrategy)
	}
	namingStrategy := reflect.ValueOf(db.NamingStrategy)
	if namingStrategy.CanConvert(gormSchemaEmbeddedNamer) {
		name = reflect.Indirect(namingStrategy.Convert(gormSchemaEmbeddedNamer)).FieldByName("Table").String()
	}
	return
}
