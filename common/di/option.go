package di

import (
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"go.uber.org/fx"
)

type provideOption struct {
	name   string
	group  string
	export bool
	as     interface{}
}

func Name(name string) utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.name = name
	}
}

func Group(group string) utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.group = group
	}
}

func Export() utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.export = true
	}
}

func As(as interface{}) utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.as = as
	}
}

type scopeOption struct {
}

type appOption struct {
	module                    string
	logOption                 fx.Option
	startTimeout, stopTimeout time.Duration
}

func StartTimeout(timeout time.Duration) utils.OptionFunc[appOption] {
	return func(a *appOption) { a.startTimeout = timeout }
}

func StopTimeout(timeout time.Duration) utils.OptionFunc[appOption] {
	return func(a *appOption) { a.stopTimeout = timeout }
}

func Scope(name string) utils.OptionFunc[appOption] {
	return func(a *appOption) { a.module = name }
}

func WithLogger(ctor any) utils.OptionFunc[appOption] {
	return func(a *appOption) {
		a.logOption = fx.WithLogger(ctor)
	}
}
