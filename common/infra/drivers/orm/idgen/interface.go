package idgen

import (
	"errors"

	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/utils"
)

var (
	ErrNewGenerator = errors.New("new id generator error")
)

type Generator interface {
	Next(opts ...utils.OptionExtender) (id uint64, err error)
}

type option struct {
	tx        *gorm.DB
	idx       int64
	tableName string
}

func GormTx(tx *gorm.DB) utils.OptionFunc[option] {
	return func(o *option) {
		o.tx = tx
	}
}
