package di

import (
	"go.uber.org/dig"

	"github.com/wfusion/gofusion/common/utils"
)

type DI interface {
	Invoke(fn any) error
	MustInvoke(fn any)
	Provide(fn any, opts ...utils.OptionExtender) error
	MustProvide(fn any, opts ...utils.OptionExtender) DI
	Decorate(decorator any) error
	MustDecorate(decorator any) DI
	String() string
	Clear()
	Preload()
}

type In struct {
	dig.In
}
type Out struct {
	dig.Out
}
