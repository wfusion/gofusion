package utils

import (
	"context"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test"
)

var (
	T = &Test{Suite: test.S}
)

type Test struct {
	*test.Suite
}

func (t *Test) SetupTest() {
	t.Catch(func() {
		log.Info(context.Background(), "------------ utils test case begin ------------")
	})
}

func (t *Test) TearDownTest() {
	t.Catch(func() {
		log.Info(context.Background(), "------------ utils test case end ------------")
	})
}
