package cases

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"

	testRoutine "github.com/wfusion/gofusion/test/routine"
)

func TestPool(t *testing.T) {
	testingSuite := &Pool{Test: testRoutine.T}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Pool struct {
	*testRoutine.Test
}

func (t *Pool) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Pool) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Pool) TestSubmit() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		pool := routine.NewPool("test_submit", 1,
			routine.WithoutTimeout(), routine.AppName(testRoutine.Component))
		defer pool.Release()

		t.NoError(pool.Submit(func() { defer wg.Done(); i += 1 }))
		wg.Wait()

		t.EqualValues(1, i)
	})
}

func (t *Pool) TestSubmitWithArgs() {
	t.Catch(func() {
		wg := new(sync.WaitGroup)
		i := 0
		wg.Add(1)
		pool := routine.NewPool("test_submit_with_args", 1,
			routine.WithoutTimeout(), routine.AppName(testRoutine.Component))
		defer pool.Release()

		t.NoError(pool.Submit(func(delta int) { defer wg.Done(); i += delta }, routine.Args(2)))
		wg.Wait()

		t.EqualValues(2, i)
	})
}
