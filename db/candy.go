package db

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

type scanOption struct {
	dbName string

	cursors       []any
	cursorWhere   any
	cursorColumns []string

	where           any
	sqlAndArguments []any

	order any

	batch int
	limit int

	log log.Loggable
}

type scanOptionGeneric[T any, TS ~[]*T] struct {
	dal DalInterface[T, TS]
}

func ScanDAL[T any, TS ~[]*T](dal DalInterface[T, TS]) utils.OptionFunc[scanOptionGeneric[T, TS]] {
	return func(o *scanOptionGeneric[T, TS]) {
		o.dal = dal
	}
}

func ScanUse(dbName string) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.dbName = dbName
	}
}

func ScanWhere(where any, sqlAndArguments ...any) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.where = where
		o.sqlAndArguments = sqlAndArguments
	}
}

func ScanCursor(cursorWhere any, cursorColumns []string, cursors ...any) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.cursors = cursors
		o.cursorWhere = cursorWhere
		o.cursorColumns = cursorColumns
	}
}

func ScanOrder(order any) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.order = order
	}
}

func ScanBatch(batch int) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.batch = batch
	}
}

func ScanLimit(limit int) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.limit = limit
	}
}

func ScanLog(log log.Loggable) utils.OptionFunc[scanOption] {
	return func(o *scanOption) {
		o.log = log
	}
}

func Scan[T any, TS ~[]*T](ctx context.Context, cb func(TS) bool, opts ...utils.OptionExtender) (err error) {
	var (
		tx    *gorm.DB
		mList TS
	)

	o := utils.ApplyOptions[useOption](opts...)
	opt := utils.ApplyOptions[scanOption](opts...)
	optG := utils.ApplyOptions[scanOptionGeneric[T, TS]](opts...)

	// get db instance
	switch {
	case optG.dal != nil:
		tx = optG.dal.ReadDB(ctx)
	case opt.dbName != "":
		tx = Use(ctx, opt.dbName, AppName(o.appName)).GetProxy()
	default:
		panic(errors.New("unknown which table to scan"))
	}

	// default values
	if opt.cursors == nil {
		opt.cursors = []any{0}
	}
	if opt.cursorWhere == nil {
		opt.cursorWhere = "id > ?"
	}
	if len(opt.cursorColumns) == 0 {
		opt.cursorColumns = []string{"id"}
	}
	if opt.order == nil {
		opt.order = fmt.Sprintf("%s ASC", strings.Join(opt.cursorColumns, constant.Comma))
	}
	if opt.batch == 0 {
		opt.batch = 100
	}
	if opt.limit == 0 {
		opt.limit = math.MaxInt
	}

	count := 0
	tx = tx.WithContext(ctx)
	if opt.log != nil {
		opt.log.Info(ctx, "scan begin [where[%s][%+v] cursor[%s][%+v] order[%s] limit[%v] batch[%v]]",
			opt.where, opt.sqlAndArguments, opt.cursorWhere, opt.cursors, opt.order, opt.limit, opt.batch)

		defer func() { opt.log.Info(ctx, "scan end [count[%v]]", count) }()
	}

	// scan
	for hasMore := true; hasMore; hasMore = len(mList) >= opt.batch {
		// init model slice
		mList = make(TS, 0, opt.batch)

		// db query
		q := tx.Where(opt.cursorWhere, opt.cursors...)
		if opt.where != nil {
			q = q.Where(opt.where, opt.sqlAndArguments...)
		}
		if opt.order != nil {
			q = q.Order(opt.order)
		}
		if err = q.Limit(opt.batch).Find(&mList).Error; err != nil {
			if opt.log != nil {
				opt.log.Warn(ctx, "scan quit because meet with error [err[%s]]", err)
			}
			break
		}

		if len(mList) > 0 {
			// callback
			if !cb(mList) {
				if opt.log != nil {
					opt.log.Info(ctx, "scan quit because callback return false")
				}
				break
			}

			// get next cursor
			next := mList[len(mList)-1]
			nextVal := reflect.Indirect(reflect.ValueOf(next))
			for idx, col := range opt.cursorColumns {
				fieldVal := nextVal.FieldByNameFunc(func(s string) bool { return strings.EqualFold(s, col) })
				if !fieldVal.IsValid() {
					fieldVal, _ = utils.GetGormColumnValue(next, col)
				}
				if !fieldVal.IsValid() {
					err = errors.Errorf("scan cursor column value is not found [col[%s]]", col)
					if opt.log != nil {
						opt.log.Error(ctx, "%s", err)
					}
					return
				}
				opt.cursors[idx] = fieldVal.Interface()
			}
		}

		// check if exceed max
		if count += len(mList); count >= opt.limit {
			if opt.log != nil {
				opt.log.Info(ctx, "scan quit because reach max [count[%v] max[%v]]", count, opt.limit)
			}
			break
		}
	}

	return
}
