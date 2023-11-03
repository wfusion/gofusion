package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test"
)

var (
	T         = &Test{Suite: test.S}
	Component = "cache"
)

type Test struct {
	*test.Suite

	once  sync.Once
	exits []func()

	testsLefts atomic.Int64
}

func (t *Test) SetupTest() {
	t.Catch(func() {
		log.Info(context.Background(), fmt.Sprintf("------------ %s test case begin ------------", Component))

		files := []string{"app.local.yml", "app.yml"}
		t.once.Do(func() {
			// t.exits = append(t.exits, t.Suite.Copy(files, 1))
			t.Cleanup(t.Suite.Copy(files, 1))
		})

		t.Cleanup(t.Suite.Init(files, 1))
		// t.exits = append(t.exits, t.Suite.Init(files, 1))
	})
}

func (t *Test) TearDownTest() {
	t.Catch(func() {
		log.Info(context.Background(), fmt.Sprintf("------------ %s test case end ------------", Component))
		if t.testsLefts.Add(-1) == 0 {
			for i := len(t.exits) - 1; i >= 0; i-- {
				t.exits[i]()
			}
		}
	})
}

func (t *Test) Init(testingSuite suite.TestingSuite) {
	methodFinder := reflect.TypeOf(testingSuite)
	numMethod := methodFinder.NumMethod()

	numTestLeft := int64(0)
	for i := 0; i < numMethod; i++ {
		method := methodFinder.Method(i)
		ok, _ := test.MethodFilter(method.Name)
		if !ok {
			continue
		}
		numTestLeft++
	}
	t.testsLefts.Add(numTestLeft)
}
