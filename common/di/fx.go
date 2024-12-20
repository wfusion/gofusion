package di

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
)

type _fx struct {
	*fx.App
	module string
	opts   []fx.Option

	mu                        sync.Mutex
	startTimeout, stopTimeout time.Duration
}

func (f *_fx) Run()                            { f.use().Run() }
func (f *_fx) Start(ctx context.Context) error { return f.use().Start(ctx) }
func (f *_fx) Stop(ctx context.Context) error  { return f.use().Stop(ctx) }
func (f *_fx) Wait() <-chan fx.ShutdownSignal  { return f.use().Wait() }
func (f *_fx) Done() <-chan os.Signal          { return f.use().Done() }

func (f *_fx) Invoke(fn any) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, fx.Invoke(fn))
	return
}

func (f *_fx) MustInvoke(fn any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, fx.Invoke(fn))
}

func (f *_fx) Provide(ctor any, opts ...utils.OptionExtender) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	opt := utils.ApplyOptions[provideOption](opts...)
	annotations := make([]fx.Annotation, 0, 3)
	if opt.name != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf(`name:"%s"`, opt.name)))
	}
	if opt.group != "" {
		annotations = append(annotations, fx.ResultTags(fmt.Sprintf(`group:"%s"`, opt.group)))
	}
	if opt.as != nil {
		annotations = append(annotations, fx.As(opt.as))
	}

	f.opts = append(f.opts, fx.Provide(fx.Annotate(ctor, annotations...)))
	return
}

func (f *_fx) MustProvide(ctor any, opts ...utils.OptionExtender) DI {
	utils.MustSuccess(f.Provide(ctor, opts...))
	return f
}

func (f *_fx) Decorate(decorator any) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, fx.Decorate(decorator))
	return
}

func (f *_fx) MustDecorate(decorator any) DI {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, fx.Decorate(decorator))
	return f
}

func (f *_fx) Populate(objs ...any) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.opts = append(f.opts, fx.Populate(objs...))
	return
}

func (f *_fx) MustPopulate(objs ...any) DI {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, fx.Populate(objs...))
	return f
}

func (f *_fx) Scope(name string, opts ...utils.OptionExtender) DI {
	f.mu.Lock()
	defer f.mu.Unlock()
	fxOpts := make([]fx.Option, 0, len(f.opts))
	for _, opt := range f.opts {
		fxOpts = append(fxOpts, opt)
	}
	return &_fx{
		App:    nil,
		module: name,
		opts:   fxOpts,
		mu:     sync.Mutex{},
	}
}

func (f *_fx) Options(opts ...fx.Option) App {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opts = append(f.opts, opts...)
	return f
}

func (f *_fx) String() string {
	app := f.use()

	f.mu.Lock()
	defer f.mu.Unlock()
	fnp := inspect.FuncOf("go.uber.org/fx.(*App).dotGraph")
	fn := *(*func(*fx.App) (fx.DotGraph, error))(fnp)
	graph, err := fn(app)
	if err != nil {
		panic(err)
	}
	return string(graph)
}

func (f *_fx) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.App = nil
	f.opts = nil
}

func (f *_fx) Preload() {}

func (f *_fx) use() *fx.App {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.App != nil {
		return f.App
	}

	opts := f.opts
	if f.module != "" {
		opts = []fx.Option{fx.Module(f.module, f.opts...)}
	}
	if f.startTimeout > 0 {
		fx.StartTimeout(f.startTimeout)
	}
	if f.stopTimeout > 0 {
		fx.StopTimeout(f.stopTimeout)
	}

	f.App = fx.New(opts...)
	return f.App
}
