package http

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	_ "unsafe"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test"

	fmkHtp "github.com/wfusion/gofusion/http"
)

var (
	T         = &Test{Suite: test.S}
	Component = "http"
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

func (t *Test) ServerGiven(method, path string, hd any) (r fmkHtp.IRouter) {
	r = newRouter(gin.New(), Component)
	switch method {
	case http.MethodGet:
		r.GET(path, hd)
	case http.MethodPost:
		r.POST(path, hd)
	case http.MethodDelete:
		r.DELETE(path, hd)
	case http.MethodPatch:
		r.PATCH(path, hd)
	case http.MethodPut:
		r.PUT(path, hd)
	case http.MethodOptions:
		r.OPTIONS(path, hd)
	case http.MethodHead:
		r.HEAD(path, hd)
	case "Any":
		r.Any(path, hd)
	case "Handle":
		r.Handle(path, hd)
	case "File":
		r.StaticFile(path, hd.(string))
	}
	return
}

//go:linkname newRouter github.com/wfusion/gofusion/http.newRouter
func newRouter(gin.IRouter, string) fmkHtp.IRouter
