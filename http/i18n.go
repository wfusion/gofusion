package http

import (
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"
)

var (
	I18n  i18n.Localizable[Errcode]
	i18ns map[string]i18n.Localizable[Errcode]
)

func Localizable(opts ...utils.OptionExtender) i18n.Localizable[Errcode] {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	i, ok := i18ns[opt.appName]
	if !ok {
		panic(errors.Errorf("http i18n not founc: %s", opt.appName))
	}
	return i
}
