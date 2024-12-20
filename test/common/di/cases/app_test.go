package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/common/di"

	comDI "github.com/wfusion/gofusion/common/di"
)

func TestApp(t *testing.T) {
	t.Parallel()
	testingSuite := &App{Test: new(di.Test)}
	suite.Run(t, testingSuite)
}

type App struct {
	*di.Test
}

func (t *App) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *App) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		comDI.Fx.Clear()
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *App) TestApp() {
	t.Catch(func() {
		comDI.Fx.
			MustProvide(NewPersonDrink).
			MustProvide(NewPersonEat[int]).
			MustProvide(NewPersonEat[string])

		comDI.Fx.MustInvoke(func(p Person) { p.Show() })
		comDI.Fx.MustInvoke(func(p Person2) { p.Show() })

		t.Require().NoError(comDI.Fx.Start(context.Background()))
		t.Require().NoError(comDI.Fx.Stop(context.Background()))
	})
}

func (t *App) TestShutdown() {
	t.Catch(func() {
		type pointer struct{}

		comDI.Fx.
			MustProvide(NewPersonDrink).
			MustProvide(NewPersonEat[bool]).
			MustProvide(NewPersonEat[string]).
			MustProvide(NewPersonEat[int]).
			MustProvide(NewPersonEat[int8]).
			MustProvide(NewPersonEat[int16]).
			MustProvide(NewPersonEat[int32]).
			MustProvide(NewPersonEat[int64]).
			MustProvide(NewPersonEat[uint]).
			MustProvide(NewPersonEat[uint8]).
			MustProvide(NewPersonEat[uint16]).
			MustProvide(NewPersonEat[uint32]).
			MustProvide(NewPersonEat[uint64]).
			MustProvide(NewPersonEat[[]int]).
			MustProvide(NewPersonEat[[]int8]).
			MustProvide(NewPersonEat[[]int16]).
			MustProvide(NewPersonEat[[]int32]).
			MustProvide(NewPersonEat[[]int64]).
			MustProvide(NewPersonEat[[]uint]).
			MustProvide(NewPersonEat[[]uint8]).
			MustProvide(NewPersonEat[[]uint16]).
			MustProvide(NewPersonEat[[]uint32]).
			MustProvide(NewPersonEat[[]uint64]).
			MustProvide(NewPersonEat[[]bool]).
			MustProvide(NewPersonEat[[]string]).
			MustProvide(func() *pointer { return new(pointer) })

		onStart := false
		onStop := false
		invokeAllFn := func(
			lc fx.Lifecycle,
			shutdown fx.Shutdowner,
			d Drink,
			e1 Eat[bool], e4 Eat[string],
			e5 Eat[int], e6 Eat[int8], e7 Eat[int16], e8 Eat[int32], e9 Eat[int64],
			e10 Eat[uint], e11 Eat[uint8], e12 Eat[uint16], e13 Eat[uint32], e14 Eat[uint64],
			e15 Eat[[]int], e16 Eat[[]int8], e17 Eat[[]int16], e18 Eat[[]int32], e19 Eat[[]int64],
			e20 Eat[[]uint], e21 Eat[[]uint8], e22 Eat[[]uint16], e23 Eat[[]uint32], e24 Eat[[]uint64],
			e25 Eat[[]bool], e28 Eat[[]string], p *pointer,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) (err error) {
					onStart = true
					println("app call start")
					return shutdown.Shutdown(fx.ExitCode(2))
				},
				OnStop: func(ctx context.Context) (err error) {
					onStop = true
					println("app call stop")
					return
				},
			})
		}
		comDI.Fx.MustInvoke(invokeAllFn)

		var s fx.Shutdowner
		comDI.Fx.MustPopulate(&s)

		ctx := context.Background()
		t.Require().NoError(comDI.Fx.Start(ctx))
		t.NoError(s.Shutdown(fx.ExitCode(0)))
		t.Require().Zero((<-comDI.Fx.Wait()).ExitCode)
		t.Require().NoError(comDI.Fx.Stop(ctx))

		t.Require().True(onStart)
		t.Require().True(onStop)
	})
}

func (t *App) TestName() {
	t.Catch(func() {
		t.NoError(comDI.Fx.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.NoError(comDI.Fx.Provide(NewPersonEat[int]))
		t.NoError(comDI.Fx.Invoke(func(p Person3) { p.Show() }))

		ctx := context.Background()
		t.NoError(comDI.Fx.Start(ctx))
		t.NoError(comDI.Fx.Stop(ctx))
	})
}

func (t *App) TestGroup() {
	t.Catch(func() {
		t.NoError(comDI.Fx.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.NoError(comDI.Fx.Provide(NewPersonEat[int]))

		t.NoError(comDI.Fx.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.NoError(comDI.Fx.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.NoError(comDI.Fx.Invoke(func(p Person4) { p.Show() }))

		ctx := context.Background()
		t.NoError(comDI.Fx.Start(ctx))
		t.NoError(comDI.Fx.Stop(ctx))
	})
}

func (t *App) TestString() {
	t.Catch(func() {
		t.NoError(comDI.Fx.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.NoError(comDI.Fx.Provide(NewPersonEat[int]))

		t.NoError(comDI.Fx.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.NoError(comDI.Fx.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.NoError(comDI.Fx.Invoke(func(p Person4) { p.Show() }))

		graph := comDI.Fx.String()
		log.Info(context.Background(), graph)
		t.NotEmpty(graph)
	})
}
