package cases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"

	testRoutine "github.com/wfusion/gofusion/test/routine"
)

func TestCandy(t *testing.T) {
	testingSuite := &Candy{Test: new(testRoutine.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Candy struct {
	*testRoutine.Test
}

func (t *Candy) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Candy) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Candy) TestGo() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(func() {
			i += 1
		}, routine.WaitGroup(wg), routine.AppName(t.AppName()))
		wg.Wait()
		t.EqualValues(1, i)
	})
}

func (t *Candy) TestGoWithArgs() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(args ...any) {
				i += args[0].(int)
			},
			routine.Args(2, 3, 4, 5),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(2, i)
	})
}

func (t *Candy) TestGoWithType() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(arg int) { i += arg },
			routine.Args(2),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(2, i)
	})
}

func (t *Candy) TestGoWithVariableArgs() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(num int, str string, args ...uint) {
				i += int(args[0])
			},
			routine.Args(2, "this is a string", 4, 5, 6),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(4, i)
	})
}

func (t *Candy) TestGoWithError() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(num int, str string, args ...uint) error {
				i += int(args[0])
				return errors.New("no")
			},
			routine.Args(2, "this is a string", 4, 5, 6),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(4, i)
	})
}

func (t *Candy) TestGoWithResultAndError() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(num int, str string, args ...uint) (any, error) {
				i += int(args[0])
				return i, nil
			},
			routine.Args(2, "this is a string", 4, 5, 6),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(4, i)
	})
}

func (t *Candy) TestGocWithResultAndError() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Goc(
			context.Background(),
			func(num int, str string, args ...uint) (any, error) {
				i += int(args[0])
				return i, errors.New("get an error")
			},
			routine.Args(2, "this is a string", 4, 5, 6),
			routine.WaitGroup(wg),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(4, i)
	})
}

func (t *Candy) TestGoWithChannel() {
	t.Catch(func() {
		ch := make(chan any, 1)
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(args ...any) {
				i += args[0].(int)
			},
			routine.Args(2, 3, 4, 5),
			routine.WaitGroup(wg),
			routine.Channel(ch),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(2, i)
		select {
		case v := <-ch:
			log.Info(context.Background(), "get channel result: %+v", v)
		}
	})
}

func (t *Candy) TestGoWithChannelResult() {
	t.Catch(func() {
		ch := make(chan any, 1)
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(args ...any) (any, error) {
				i += args[0].(int)
				return i, nil
			},
			routine.Args(2, 3, 4, 5),
			routine.WaitGroup(wg),
			routine.Channel(ch),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(2, i)
		select {
		case v := <-ch:
			log.Info(context.Background(), "get channel result: %+v", v)
		}
	})
}

func (t *Candy) TestGoWithChannelError() {
	t.Catch(func() {
		ch := make(chan any, 1)
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		routine.Go(
			func(args ...any) (any, error) {
				i += args[0].(int)
				return i, errors.New("channel error")
			},
			routine.Args(2, 3, 4, 5),
			routine.WaitGroup(wg),
			routine.Channel(ch),
			routine.AppName(t.AppName()),
		)
		wg.Wait()
		t.EqualValues(2, i)
		select {
		case v := <-ch:
			log.Info(context.Background(), "get channel result: %+v", v)
		}
	})
}

func (t *Candy) TestLoop() {
	t.Catch(func() {
		i := 0
		expected := 10
		wg := new(sync.WaitGroup)
		wg.Add(1)
		routine.Loop(func() {
			for i = 0; i < expected; i++ {
			}
		}, routine.WaitGroup(wg), routine.AppName(t.AppName()))
		wg.Wait()
		t.EqualValues(expected, i)
	})
}

func (t *Candy) TestLoopWithArgs() {
	t.Catch(func() {
		i := 0
		expected := 10
		wg := new(sync.WaitGroup)
		wg.Add(1)
		routine.Loop(func(args ...any) {
			for i := args[0].(*int); *i < expected; *i++ {
			}
		}, routine.Args(&i), routine.WaitGroup(wg), routine.AppName(t.AppName()))
		wg.Wait()
		t.Equal(expected, i)
	})
}

func (t *Candy) TestWhenAllWithFuture() {
	t.Catch(func() {
		sum, expected := atomic.NewInt64(0), 10
		futures := make([]any, 0, 10)
		for i := 0; i < expected; i++ {
			future := routine.Promise(
				func(args ...any) {
					sum.Add(cast.ToInt64(args[0]))
				},
				true,
				routine.Args(2, 3, 4, 5),
				routine.AppName(t.AppName()),
			)

			futures = append(futures, future)
		}

		futures = append(futures, routine.AppName(t.AppName()))
		_, timeout, err := routine.WhenAll(futures...).GetOrTimeout(time.Second)
		t.False(timeout)
		t.NoError(err)
		t.EqualValues(expected*2, sum.Load())
	})
}
