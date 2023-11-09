package cases

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/common/di"

	comDI "github.com/wfusion/gofusion/common/di"
)

func TestExample(t *testing.T) {
	t.Parallel()
	testingSuite := &Example{Test: new(di.Test)}
	suite.Run(t, testingSuite)
}

type Example struct {
	*di.Test
}

func (t *Example) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Example) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		comDI.Dig.Clear()
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Example) TestDI() {
	t.Catch(func() {
		t.NoError(comDI.Dig.Provide(NewPersonDrink))
		t.NoError(comDI.Dig.Provide(NewPersonEat[int]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[string]))

		t.NoError(
			comDI.Dig.Invoke(func(p Person) {
				p.Show()
			}),
		)
		t.NoError(
			comDI.Dig.Invoke(func(p Person2) {
				p.Show()
			}),
		)
	})
}

func (t *Example) TestConcurrentInvoke() {
	t.Catch(func() {
		type pointer struct{}

		t.NoError(comDI.Dig.Provide(NewPersonDrink))

		t.NoError(comDI.Dig.Provide(NewPersonEat[bool]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[string]))

		t.NoError(comDI.Dig.Provide(NewPersonEat[int]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[int8]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[int16]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[int32]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[int64]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[uint]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[uint8]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[uint16]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[uint32]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[uint64]))

		t.NoError(comDI.Dig.Provide(NewPersonEat[[]int]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]int8]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]int16]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]int32]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]int64]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]uint]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]uint8]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]uint16]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]uint32]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]uint64]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]bool]))
		t.NoError(comDI.Dig.Provide(NewPersonEat[[]string]))
		t.NoError(comDI.Dig.Provide(func() *pointer { return new(pointer) }))

		comDI.Dig.Preload()

		invokeAllFn := func(
			d Drink,
			e1 Eat[bool], e4 Eat[string],
			e5 Eat[int], e6 Eat[int8], e7 Eat[int16], e8 Eat[int32], e9 Eat[int64],
			e10 Eat[uint], e11 Eat[uint8], e12 Eat[uint16], e13 Eat[uint32], e14 Eat[uint64],
			e15 Eat[[]int], e16 Eat[[]int8], e17 Eat[[]int16], e18 Eat[[]int32], e19 Eat[[]int64],
			e20 Eat[[]uint], e21 Eat[[]uint8], e22 Eat[[]uint16], e23 Eat[[]uint32], e24 Eat[[]uint64],
			e25 Eat[[]bool], e28 Eat[[]string], p *pointer,
		) {
		}

		wg := new(sync.WaitGroup)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); t.NoError(comDI.Dig.Invoke(invokeAllFn)) }()
		}

		wg.Wait()
	})
}
