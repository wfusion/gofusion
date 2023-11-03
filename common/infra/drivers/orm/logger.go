package orm

import (
	"gorm.io/gorm/logger"

	"github.com/wfusion/gofusion/common/utils"
)

func WithLogger(l logger.Interface) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.logger = l
	}
}
