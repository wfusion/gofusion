package remoteconfig

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test"
)

var (
	component = "remoteconfig"
)

type Test struct {
	test.Suite

	once  sync.Once
	exits []func()

	testName   string
	testsLefts atomic.Int64
}

func (t *Test) SetupTest() {
	t.Catch(func() {
		log.Info(context.Background(), fmt.Sprintf("------------ %s test case begin ------------", component))

		t.once.Do(func() {
			t.exits = append(t.exits, t.Suite.Copy(t.AllConfigFiles(), t.testName, 1))
		})
	})
}

func (t *Test) TearDownTest() {
	t.Catch(func() {
		log.Info(context.Background(), fmt.Sprintf("------------ %s test case end ------------", component))
		if t.testsLefts.Add(-1) == 0 {
			for i := len(t.exits) - 1; i >= 0; i-- {
				t.exits[i]()
			}
		}
	})
}

func (t *Test) ConfigFiles() []string {
	files := []string{
		"app.local.yml",
		"app.yml",
		"app.json",
		"app.toml",
	}
	for i := 0; i < len(files); i++ {
		files[i] = env.WorkDir + "/configs/" + t.AppName() + "." + files[i]
	}
	return files
}

func (t *Test) AppName() string {
	return fmt.Sprintf("%s.%s", component, t.testName)
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
	t.testName = utils.IndirectType(methodFinder).Name()
	t.testsLefts.Add(numTestLeft)
}
