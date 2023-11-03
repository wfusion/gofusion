package config

import (
	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
)

type Configurable interface {
	Init(businessConfig any, opts ...utils.OptionExtender) (gracefully func())
	LoadComponentConfig(name string, componentConfig any) (err error)
	GetAllConfigs() any
	Debug() (debug bool)
	AppName() (name string)
	DI() di.DI
}

type InitOption struct {
	DI      di.DI
	AppName string
}

func AppName(name string) utils.OptionFunc[InitOption] {
	return func(o *InitOption) {
		o.AppName = name
	}
}

func DI(di di.DI) utils.OptionFunc[InitOption] {
	return func(o *InitOption) {
		o.DI = di
	}
}
