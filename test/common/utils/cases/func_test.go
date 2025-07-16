package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/test/internal/mock"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestFunc(t *testing.T) {
	t.Parallel()
	testingSuite := &Func{Test: new(testUtl.Test)}
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
			t.Require().EqualValues(ctx, paramCtx)
			t.Require().EqualValues(obj.Basic.Str, paramObj.Basic.Str)
		}
		wrapper := utils.WrapFunc1[error](hd)

		// When
		err := wrapper(ctx, obj)
		t.Require().NoError(err)
	})
}
