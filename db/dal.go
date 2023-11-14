package db

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/db/plugins"

	ormDrv "github.com/wfusion/gofusion/common/infra/drivers/orm"
	fusCtx "github.com/wfusion/gofusion/context"
)

// DalInterface
//nolint: revive // interface issue
type DalInterface[T any, TS ~[]*T] interface {
	Query(ctx context.Context, query any, args ...any) (TS, error)
	QueryFirst(ctx context.Context, query any, args ...any) (*T, error)
	QueryLast(ctx context.Context, query any, args ...any) (*T, error)
	QueryInBatches(ctx context.Context, batchSize int, fc func(tx *DB, batch int, found TS) error, query any, args ...any) error
	Count(ctx context.Context, query any, args ...any) (int64, error)
	Pluck(ctx context.Context, column string, dest any, query any, args ...any) error
	Take(ctx context.Context, dest any, conds ...any) error
	InsertOne(ctx context.Context, mod *T, opts ...utils.OptionExtender) error
	InsertInBatches(ctx context.Context, modList TS, batchSize int, opts ...utils.OptionExtender) error
	Save(ctx context.Context, mod any, opts ...utils.OptionExtender) error
	Update(ctx context.Context, column string, value any, query any, args ...any) (int64, error)
	Updates(ctx context.Context, columns map[string]any, query any, args ...any) (int64, error)
	Delete(ctx context.Context, query any, args ...any) (int64, error)
	FirstOrCreate(ctx context.Context, mod *T, conds ...any) (int64, error)
	Transaction(ctx context.Context, fc func(tx context.Context) error, opts ...utils.OptionExtender) error
	ReadDB(ctx context.Context) *gorm.DB
	WriteDB(ctx context.Context) *gorm.DB
	SetCtxReadDB(src context.Context) (dst context.Context)
	SetCtxWriteDB(src context.Context) (dst context.Context)
	Model() *T
	ModelSlice() TS
	IgnoreErr(err error) error
	CanIgnore(err error) bool
	ShardingByValues(ctx context.Context, src []map[string]any) (dst map[string][]map[string]any, err error)
	ShardingIDGen(ctx context.Context) (id uint64, err error)
	ShardingIDListGen(ctx context.Context, amount int) (idList []uint64, err error)
	ShardingByModelList(ctx context.Context, src TS) (dst map[string]TS, err error)
}

type dal[T any, TS ~[]*T] struct {
	appName     string
	readDBName  string
	writeDBName string
}

func NewDAL[T any, TS ~[]*T](readDBName, writeDBName string, opts ...utils.OptionExtender) DalInterface[T, TS] {
	instance := new(T)
	if _, ok := any(instance).(schema.Tabler); !ok {
		panic(errors.Errorf("model unimplement schema.Tabler [model[%T] read_db[%s] write_db[%s]]",
			instance, readDBName, writeDBName))
	}
	opt := utils.ApplyOptions[useOption](opts...)
	return &dal[T, TS]{
		appName:     opt.appName,
		readDBName:  readDBName,
		writeDBName: writeDBName,
	}
}

func (d *dal[T, TS]) Query(ctx context.Context, query any, args ...any) (TS, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	found := d.ModelSlice()
	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).Find(&found)
	if d.CanIgnore(result.Error) {
		return nil, nil
	}
	return found, d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) QueryLast(ctx context.Context, query any, args ...any) (*T, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	found := d.Model()
	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).Last(found)
	if d.CanIgnore(result.Error) {
		return nil, nil
	}
	return found, d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) QueryFirst(ctx context.Context, query any, args ...any) (*T, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	found := d.Model()
	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).First(found)
	if d.CanIgnore(result.Error) {
		return nil, nil
	}
	return found, d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) QueryInBatches(ctx context.Context, batchSize int,
	fc func(tx *DB, batch int, found TS) error, query any, args ...any) (err error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	orm := Use(ctx, d.readDBName, AppName(d.appName))
	found := make(TS, 0, batchSize)
	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).FindInBatches(&found, batchSize,
		func(tx *gorm.DB, batch int) error {
			wrapper := &DB{
				DB:                   &ormDrv.DB{DB: tx},
				Name:                 orm.Name,
				tableShardingPlugins: orm.tableShardingPlugins,
			}
			return fc(wrapper, batch, found)
		},
	)
	if d.CanIgnore(result.Error) {
		return
	}
	return d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) Count(ctx context.Context, query any, args ...any) (int64, error) {
	var count int64

	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).Count(&count)
	if d.CanIgnore(result.Error) {
		return 0, nil
	}
	return count, d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) Pluck(ctx context.Context, column string, dest any,
	query any, args ...any) error {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	result := d.ReadDB(ctx).Clauses(o.clauses...).Where(query, args...).Pluck(column, dest)
	return d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) Take(ctx context.Context, dest any, conds ...any) error {
	o, args := d.parseOptionFromArgs(conds...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)

	result := d.ReadDB(ctx).Clauses(o.clauses...).Take(dest, args...)
	return d.IgnoreErr(result.Error)
}

func (d *dal[T, TS]) InsertOne(ctx context.Context, mod *T, opts ...utils.OptionExtender) error {
	o := utils.ApplyOptions[mysqlDALOption](opts...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	return d.WriteDB(ctx).Clauses(o.clauses...).Create(mod).Error
}

func (d *dal[T, TS]) InsertInBatches(ctx context.Context,
	modList TS, batchSize int, opts ...utils.OptionExtender) error {
	o := utils.ApplyOptions[mysqlDALOption](opts...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	sharded, err := d.writeWithTableSharding(ctx, modList)
	if err != nil {
		return err
	}
	for _, mList := range sharded {
		if err = d.WriteDB(ctx).Clauses(o.clauses...).CreateInBatches(mList, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}

func (d *dal[T, TS]) FirstOrCreate(ctx context.Context, mod *T, conds ...any) (int64, error) {
	o, conds := d.parseOptionFromArgs(conds...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	result := d.WriteDB(ctx).Clauses(o.clauses...).FirstOrCreate(mod, conds...)
	return result.RowsAffected, result.Error
}

// Save create or update model
// Only support for passing in *mod, []*mod, [...]*mod, it's recommended to only use *mod to call this method.
// If using mod, []mod, since it's value passing, the upper layer will not be able to
// obtain the auto-incremented id from create or other fields filled in by the lower layer.
// If using [...]mod, it will trigger panic: using unaddressable error.
// In official usage, both mod and [...]mod will trigger panic: using unaddressable error.
func (d *dal[T, TS]) Save(ctx context.Context, mod any, opts ...utils.OptionExtender) error {
	// Translate the struct to slice to follow the insert into with ON DUPLICATE KEY UPDATE
	mList, ok := d.convertAnyToTS(mod)
	if !ok {
		mList = utils.SliceConvert(mod, reflect.TypeOf(TS{})).(TS)
	}
	if len(mList) == 0 {
		return nil
	}
	o := utils.ApplyOptions[mysqlDALOption](opts...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	sharded, err := d.writeWithTableSharding(ctx, mList)
	if err != nil {
		return err
	}
	for _, mList := range sharded {
		if err = d.WriteDB(ctx).Clauses(o.clauses...).Save(mList).Error; err != nil {
			return err
		}
	}

	return nil
}

func (d *dal[T, TS]) Update(ctx context.Context, column string, value any,
	query any, args ...any) (int64, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	u := d.WriteDB(ctx).Clauses(o.clauses...).Where(query, args...).Update(column, value)
	return u.RowsAffected, u.Error
}

func (d *dal[T, TS]) Updates(ctx context.Context, columns map[string]any,
	query any, args ...any) (int64, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	u := d.WriteDB(ctx).Clauses(o.clauses...).Where(query, args...).Updates(columns)
	return u.RowsAffected, u.Error
}

func (d *dal[T, TS]) Delete(ctx context.Context, query any, args ...any) (int64, error) {
	o, args := d.parseOptionFromArgs(args...)
	ctx = context.WithValue(ctx, fusCtx.KeyDALOption, o)
	mList, ok := d.convertAnyToTS(query)
	if !ok || len(mList) == 0 {
		deleted := d.WriteDB(ctx).Clauses(o.clauses...).Where(query, args...).Delete(d.Model())
		return deleted.RowsAffected, deleted.Error
	} else {
		sharded, err := d.writeWithTableSharding(ctx, mList)
		if err != nil {
			return 0, err
		}
		var rowAffected int64
		for _, mList := range sharded {
			deleted := d.WriteDB(ctx).Clauses(o.clauses...).Delete(mList, args...)
			if deleted.Error != nil {
				return rowAffected, deleted.Error
			}
			rowAffected += deleted.RowsAffected
		}
		return rowAffected, nil
	}
}

func (d *dal[T, TS]) Transaction(ctx context.Context, fc func(context.Context) error,
	opts ...utils.OptionExtender) error {
	orm := GetCtxGormDB(ctx)
	o := utils.ApplyOptions[mysqlDALOption](opts...)
	if orm == nil || (orm.Name != d.writeDBName && orm.Name != d.readDBName) {
		if o.useWriteDB {
			orm = Use(ctx, d.writeDBName, AppName(d.appName))
		} else {
			orm = Use(ctx, d.readDBName, AppName(d.appName))
		}
	}

	return d.unscopedGormDB(orm.GetProxy().WithContext(ctx), o).Transaction(func(tx *gorm.DB) error {
		return fc(SetCtxGormDB(ctx, &DB{
			DB:                   &ormDrv.DB{DB: tx},
			Name:                 orm.Name,
			tableShardingPlugins: orm.tableShardingPlugins,
		}))
	})
}

func (d *dal[T, TS]) ReadDB(ctx context.Context) *gorm.DB {
	o, _ := ctx.Value(fusCtx.KeyDALOption).(*mysqlDALOption)
	if orm := GetCtxGormDB(ctx); orm != nil && orm.Name == d.readDBName {
		return d.unscopedGormDB(orm.Model(d.Model()), o)
	}

	dbName := d.readDBName
	if o != nil && o.useWriteDB {
		dbName = d.writeDBName
	}
	return d.unscopedGormDB(Use(ctx, dbName, AppName(d.appName)).WithContext(ctx).Model(d.Model()), o)
}
func (d *dal[T, TS]) WriteDB(ctx context.Context) *gorm.DB {
	o, _ := ctx.Value(fusCtx.KeyDALOption).(*mysqlDALOption)
	if orm := GetCtxGormDB(ctx); orm != nil && orm.Name == d.writeDBName {
		return d.unscopedGormDB(orm.Model(d.Model()), o)
	}

	return d.unscopedGormDB(Use(ctx, d.writeDBName, AppName(d.appName)).WithContext(ctx).Model(d.Model()), o)
}
func (d *dal[T, TS]) SetCtxReadDB(src context.Context) (dst context.Context) {
	if orm := GetCtxGormDB(src); orm != nil && orm.Name == d.readDBName {
		return src
	}

	return SetCtxGormDB(src, Use(src, d.readDBName, AppName(d.appName)))
}
func (d *dal[T, TS]) SetCtxWriteDB(src context.Context) (dst context.Context) {
	if orm := GetCtxGormDB(src); orm != nil && orm.Name == d.writeDBName {
		return src
	}
	return SetCtxGormDB(src, Use(src, d.writeDBName, AppName(d.appName)))
}

func (d *dal[T, TS]) Model() *T      { return new(T) }
func (d *dal[T, TS]) ModelSlice() TS { return make(TS, 0) }
func (d *dal[T, TS]) IgnoreErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}
func (d *dal[T, TS]) CanIgnore(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (d *dal[T, TS]) ShardingByValues(ctx context.Context, src []map[string]any) (
	dst map[string][]map[string]any, err error) {
	writeDB := d.writeDB(ctx)
	tableName := d.tableName(writeDB, new(T))
	tableShardingPlugin, ok := writeDB.tableShardingPlugins[tableName]
	if !ok {
		return map[string][]map[string]any{tableName: src}, nil
	}
	return tableShardingPlugin.ShardingByValues(ctx, src)
}
func (d *dal[T, TS]) ShardingIDGen(ctx context.Context) (id uint64, err error) {
	writeDB := d.writeDB(ctx)
	tableName := d.tableName(writeDB, new(T))
	tableShardingPlugin, ok := writeDB.tableShardingPlugins[tableName]
	if !ok {
		return 0, plugins.ErrIDGeneratorNotFound
	}
	return tableShardingPlugin.ShardingIDGen(ctx)
}
func (d *dal[T, TS]) ShardingIDListGen(ctx context.Context, amount int) (idList []uint64, err error) {
	writeDB := d.writeDB(ctx)
	tableName := d.tableName(writeDB, new(T))
	tableShardingPlugin, ok := writeDB.tableShardingPlugins[tableName]
	if !ok {
		return nil, plugins.ErrIDGeneratorNotFound
	}
	idList = make([]uint64, 0, amount)
	for i := 0; i < amount; i++ {
		id, err := tableShardingPlugin.ShardingIDGen(ctx)
		if err != nil {
			return nil, err
		}
		idList = append(idList, id)
	}
	return
}
func (d *dal[T, TS]) ShardingByModelList(ctx context.Context, src TS) (dst map[string]TS, err error) {
	if len(src) == 0 {
		return make(map[string]TS), nil
	}
	writeDB := d.writeDB(ctx)
	tableName := d.tableName(writeDB, src[0])
	shardingPlugin, ok := writeDB.tableShardingPlugins[tableName]
	if !ok {
		return map[string]TS{tableName: src}, nil
	}
	sharded, err := shardingPlugin.ShardingByModelList(ctx, utils.SliceMapping(src, func(t *T) any { return t })...)
	if err != nil {
		return
	}
	dst = make(map[string]TS, len(sharded))
	for suffix, item := range sharded {
		shardingTableName := tableName + suffix
		dst[shardingTableName] = TS(utils.SliceMapping(item, func(t any) *T { return t.(*T) }))
	}
	return
}

func (d *dal[T, TS]) writeDB(ctx context.Context) *DB {
	if orm := GetCtxGormDB(ctx); orm != nil && orm.Name == d.writeDBName {
		return orm
	}

	return Use(ctx, d.writeDBName, AppName(d.appName))
}
func (d *dal[T, TS]) writeWithTableSharding(ctx context.Context, src TS) (dst []TS, err error) {
	if len(src) == 0 {
		return
	}
	writeDB := d.writeDB(ctx)
	shardingPlugin, ok := writeDB.tableShardingPlugins[d.tableName(writeDB, src[0])]
	if !ok {
		return []TS{src}, nil
	}

	sharded, err := shardingPlugin.ShardingByModelList(ctx, utils.SliceMapping(src, func(t *T) any { return t })...)
	if err != nil {
		return
	}
	for _, item := range sharded {
		dst = append(dst, utils.SliceMapping(item, func(t any) *T { return t.(*T) }))
	}
	return
}
func (d *dal[T, TS]) tableName(db *DB, mod *T) (name string) {
	if tabler, ok := any(mod).(schema.Tabler); ok {
		name = tabler.TableName()
	}
	if tabler, ok := any(mod).(schema.TablerWithNamer); ok {
		name = tabler.TableName(db.NamingStrategy)
	}
	// TODO: check if embeddedNamer valid
	embeddedNamer := inspect.TypeOf("gorm.io/gorm/schema.embeddedNamer")
	namingStrategy := reflect.ValueOf(db.NamingStrategy)
	if namingStrategy.CanConvert(embeddedNamer) {
		name = namingStrategy.Convert(embeddedNamer).FieldByName("Table").String()
	}
	return
}
func (d *dal[T, TS]) convertAnyToTS(query any) (mList TS, ok bool) {
	switch q := query.(type) {
	case TS:
		ok = true
		mList = q
	case []*T:
		ok = true
		mList = TS(q)
	case []T:
		ok = true
		mList = TS(utils.SliceMapping(q, func(t T) *T { return &t }))
	case T:
		ok = true
		mList = TS{&q}
	case *T:
		ok = true
		mList = TS{q}
	}
	return
}
func (d *dal[T, TS]) unscopedGormDB(src *gorm.DB, o *mysqlDALOption) (dst *gorm.DB) {
	if o != nil && o.unscoped {
		return src.Unscoped()
	}
	return src
}

type mysqlDALOption struct {
	unscoped   bool
	useWriteDB bool
	clauses    []clause.Expression
}

func Unscoped() utils.OptionFunc[mysqlDALOption] {
	return func(m *mysqlDALOption) {
		m.unscoped = true
	}
}

func Clauses(clauses ...clause.Expression) utils.OptionFunc[mysqlDALOption] {
	return func(m *mysqlDALOption) {
		m.clauses = append(m.clauses, clauses...)
	}
}

func WriteDB() utils.OptionFunc[mysqlDALOption] {
	return func(m *mysqlDALOption) {
		m.useWriteDB = true
	}
}

func (d *dal[T, TS]) parseOptionFromArgs(args ...any) (o *mysqlDALOption, r []any) {
	o = new(mysqlDALOption)
	r = make([]any, 0, len(args))
	for _, arg := range args {
		if reflect.TypeOf(arg).Implements(gormClauseExpressionType) {
			o.clauses = append(o.clauses, arg.(clause.Expression))
			continue
		}

		switch v := arg.(type) {
		case utils.OptionFunc[mysqlDALOption]:
			v(o)
		default:
			r = append(r, arg)
		}
	}
	return
}
