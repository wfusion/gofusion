package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/mock"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestFunc(t *testing.T) {
	testingSuite := &Func{Test: testUtl.T}
	suite.Run(t, testingSuite)
}

type Func struct {
	*testUtl.Test
}

func (t *Func) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Func) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Func) TestWrapFuncMissInAndOutMatch() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		obj := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob).(*mock.RandomObj)
		hd := func(paramCtx context.Context, paramObj *mock.CommonObj) {
			t.EqualValues(ctx, paramCtx)
			t.EqualValues(obj.Basic.Str, paramObj.Basic.Str)
		}
		wrapper := utils.WrapFunc1[error](hd)

		// When
		err := wrapper(ctx, obj)
		t.NoError(err)
	})
}
