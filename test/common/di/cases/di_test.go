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

func TestDI(t *testing.T) {
	t.Parallel()
	testingSuite := &DI{Test: new(di.Test)}
	suite.Run(t, testingSuite)
}

type DI struct {
	*di.Test
}

func (t *DI) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *DI) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		comDI.Dig.Clear()
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *DI) TestDI() {
	t.Catch(func() {
		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[string]))

		t.Require().NoError(
			comDI.Dig.Invoke(func(p Person) {
				p.Show()
			}),
		)
		t.Require().NoError(
			comDI.Dig.Invoke(func(p Person2) {
				p.Show()
			}),
		)
	})
}

func (t *DI) TestName() {
	t.Catch(func() {
		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))
		t.Require().NoError(comDI.Dig.Invoke(func(p Person3) {
			p.Show()
		}))
	})
}

func (t *DI) TestGroup() {
	t.Catch(func() {
		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))

		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.Require().NoError(comDI.Dig.Invoke(func(p Person4) {
			p.Show()
		}))
	})
}

func (t *DI) TestPopulate() {
	t.Catch(func() {
		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))

		var d Drink
		t.Require().NoError(comDI.Dig.Populate(&d))
		d.Water()
	})
}

func (t *DI) TestString() {
	t.Catch(func() {
		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink, comDI.Name("ddd")))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))

		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int], comDI.Group("aaa")))
		t.Require().NoError(comDI.Dig.Invoke(func(p Person4) {
			p.Show()
		}))

		graph := comDI.Dig.String()
		log.Info(context.Background(), graph)
		t.Require().NotEmpty(graph)
	})
}

func (t *DI) TestConcurrentInvoke() {
	t.Catch(func() {
		type pointer struct{}

		t.Require().NoError(comDI.Dig.Provide(NewPersonDrink))

		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[bool]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[string]))

		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int8]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int16]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int32]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[int64]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[uint]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[uint8]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[uint16]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[uint32]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[uint64]))

		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]int]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]int8]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]int16]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]int32]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]int64]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]uint]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]uint8]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]uint16]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]uint32]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]uint64]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]bool]))
		t.Require().NoError(comDI.Dig.Provide(NewPersonEat[[]string]))
		t.Require().NoError(comDI.Dig.Provide(func() *pointer { return new(pointer) }))

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
			go func() { defer wg.Done(); t.Require().NoError(comDI.Dig.Invoke(invokeAllFn)) }()
		}

		wg.Wait()
	})
}
