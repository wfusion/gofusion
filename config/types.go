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
	App() di.App
}

type InitOption struct {
	DI      di.DI
	App     di.App
	AppName string
}

func AppName(name string) utils.OptionFunc[InitOption] {
	return func(o *InitOption) {
		o.AppName = name
	}
}

func App(app di.App) utils.OptionFunc[InitOption] {
	return func(o *InitOption) {
		o.App = app
	}
}

func DI(di di.DI) utils.OptionFunc[InitOption] {
	return func(o *InitOption) {
		o.DI = di
	}
}
