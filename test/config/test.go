package config

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test"
)

var (
	T         = &Test{Suite: test.S}
	Component = "config"
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

		files := []string{
			"app.local.yml",
			"app.yml",
			"app.json",
			"app.toml",
		}
		t.once.Do(func() {
			// t.exits = append(t.exits, t.Suite.Copy(files, 1))
			t.Cleanup(t.Suite.Copy(files, 1))
		})
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

func (t *Test) ConfigFiles() []string {
	files := []string{"config.app.local.yml", "config.app.yml"}
	for i := 0; i < len(files); i++ {
		files[i] = env.WorkDir + "/configs/" + files[i]
	}
	return files
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
