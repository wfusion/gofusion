package di

import (
	"context"
	"os"

	"go.uber.org/dig"
	"go.uber.org/fx"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
)

var (
	Dig = NewDI()
	Fx  = New()
)

type DI interface {
	Invoke(fn any) error
	MustInvoke(fn any)
	Provide(fn any, opts ...utils.OptionExtender) error
	MustProvide(fn any, opts ...utils.OptionExtender) DI
	Decorate(decorator any) error
	MustDecorate(decorator any) DI
	Populate(objs ...any) error
	MustPopulate(objs ...any) DI
	Scope(name string, opts ...utils.OptionExtender) DI
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

func NewDI() DI {
	return &_dig{scope: inspect.GetField[*dig.Scope](dig.New(), "scope")}
}

type App interface {
	DI
	Run()
	Options(opts ...fx.Option) App
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Wait() <-chan fx.ShutdownSignal
	Done() <-chan os.Signal
}

func New(opts ...utils.OptionExtender) App {
	opt := utils.ApplyOptions[appOption](opts...)
	app := &_fx{module: opt.module, startTimeout: opt.startTimeout, stopTimeout: opt.stopTimeout}
	if opt.logOption != nil {
		app.opts = append(app.opts, opt.logOption)
	}
	return app
}
