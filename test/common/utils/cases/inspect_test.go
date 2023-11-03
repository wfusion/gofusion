package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm/schema"

	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestInspect(t *testing.T) {
	testingSuite := &Inspect{Test: testUtl.T}
	suite.Run(t, testingSuite)
}

type Inspect struct {
	*testUtl.Test
}

func (t *Inspect) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Inspect) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Inspect) TestSetField() {
	t.Catch(func() {
		type aStruct struct {
			aa    int
			namer schema.Namer
		}

		a := &aStruct{
			aa:    1,
			namer: nil,
		}
		inspect.SetField(a, "aa", 2)
		inspect.SetField(a, "namer", schema.Namer(schema.NamingStrategy{}))

		t.Require().EqualValues(2, a.aa)
		t.Require().NotNil(a.namer)
	})
}

func (t *Inspect) TestFuncOf() {
	t.Catch(func() {
		type beforeTest func(t *Inspect, suiteName, testName string)

		fnp := inspect.FuncOf("github.com/wfusion/gofusion/test/common/utils/cases.(*Inspect).BeforeTest")
		if fnp == nil {
			t.FailNow("function not found")
			return
		}

		fn := *(*beforeTest)(fnp)
		fn(t, "inspect", "funcof")
	})
}
