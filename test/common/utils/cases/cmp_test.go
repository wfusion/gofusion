package cases

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/cmp"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestCmp(t *testing.T) {
	t.Parallel()
	testingSuite := &Cmp{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Cmp struct {
	*testUtl.Test
}

func (t *Cmp) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Cmp) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Cmp) TestComparable() {
	t.Catch(func() {
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(1), utils.AnyPtr(1)))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(1), utils.AnyPtr(2)))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(math.MaxInt), utils.AnyPtr(math.MaxInt)))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(math.MinInt), utils.AnyPtr(math.MinInt)))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int8(1)), utils.AnyPtr(int8(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(int8(1)), utils.AnyPtr(int8(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int8(math.MaxInt8)), utils.AnyPtr(int8(math.MaxInt8))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int8(math.MinInt8)), utils.AnyPtr(int8(math.MinInt8))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int16(1)), utils.AnyPtr(int16(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(int16(1)), utils.AnyPtr(int16(2))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int16(math.MaxInt16)), utils.AnyPtr(int16(math.MaxInt16))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int16(math.MinInt16)), utils.AnyPtr(int16(math.MinInt16))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int32(1)), utils.AnyPtr(int32(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(int32(1)), utils.AnyPtr(int32(2))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int32(math.MaxInt32)), utils.AnyPtr(int32(math.MaxInt32))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int32(math.MinInt32)), utils.AnyPtr(int32(math.MinInt32))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(int64(1)), utils.AnyPtr(int64(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(int64(1)), utils.AnyPtr(int64(2))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int64(math.MaxInt64)), utils.AnyPtr(int64(math.MaxInt64))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(int64(math.MinInt64)), utils.AnyPtr(int64(math.MinInt64))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint(1)), utils.AnyPtr(uint(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(uint(1)), utils.AnyPtr(uint(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint(math.MaxUint)), utils.AnyPtr(uint(math.MaxUint))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint(0)), utils.AnyPtr(uint(0))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint8(1)), utils.AnyPtr(uint8(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(uint8(1)), utils.AnyPtr(uint8(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint8(0)), utils.AnyPtr(uint8(0))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(uint8(math.MaxUint8)), utils.AnyPtr(uint8(math.MaxUint8))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint16(1)), utils.AnyPtr(uint16(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(uint16(1)), utils.AnyPtr(uint16(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint16(0)), utils.AnyPtr(uint16(0))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(uint16(math.MaxUint16)), utils.AnyPtr(uint16(math.MaxUint16))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint32(1)), utils.AnyPtr(uint32(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(uint32(1)), utils.AnyPtr(uint32(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint32(0)), utils.AnyPtr(uint32(0))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(uint32(math.MaxUint32)), utils.AnyPtr(uint32(math.MaxUint32))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint64(1)), utils.AnyPtr(uint64(1))))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(uint64(1)), utils.AnyPtr(uint64(2))))
		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(uint64(0)), utils.AnyPtr(uint64(0))))
		t.Require().Equal(true,
			cmp.ComparablePtr(utils.AnyPtr(uint64(math.MaxUint64)), utils.AnyPtr(uint64(math.MaxUint64))))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr("1"), utils.AnyPtr("1")))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr("1"), utils.AnyPtr("12")))

		t.Require().Equal(true, cmp.ComparablePtr(utils.AnyPtr(true), utils.AnyPtr(true)))
		t.Require().Equal(false, cmp.ComparablePtr(utils.AnyPtr(true), utils.AnyPtr(false)))

	})
}

func (t *Cmp) TestMapAny() {
	t.Catch(func() {
		base := map[string]any{
			"1": map[string]any{
				"2": []int{1, 2, 3},
				"3": utils.AnyPtr("3"),
				"4": []byte("123"),
				"5": map[string]any{
					"6": 1e2,
				},
			},
			"2": 2,
			"3": []any{1, complex(1, 2), 1.0, uint(2)},
		}
		a := clone.Slowly(base)
		b := clone.Slowly(base)
		t.Require().True(cmp.MapAny(a, b))

		(b["1"].(map[string]any))["5"].(map[string]any)["6"] = 1e3
		t.False(cmp.MapAny(a, b))
	})
}
