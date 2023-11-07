package cases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestClone(t *testing.T) {
	t.Parallel()
	testingSuite := &Clone{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Clone struct {
	*testUtl.Test
}

func (t *Clone) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Clone) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Clone) TestUnsafeAny() {
	t.Catch(func() {
		type cases struct {
			name string
			src  any
		}
		given := []cases{
			{
				name: "basic",
				src: struct {
					Str       string
					Int       int
					Int64     int64
					Uint      uint
					Uint64    uint64
					Float64   float64
					Complex64 complex64
					Bool      bool
					Now       time.Time
					Loc       *time.Location
				}{
					Str:       "1",
					Int:       1,
					Int64:     1,
					Uint:      1,
					Uint64:    1,
					Float64:   1,
					Complex64: complex64(1),
					Bool:      true,
					Now:       time.Now(),
					Loc:       utils.Must(time.LoadLocation("Asia/Shanghai")),
				},
			},
		}

		for _, cs := range given {
			t.Run(cs.name, func() {
				dst := clone.Clone(cs.src)
				t.Require().EqualValues(cs.src, dst)
			})
		}
	})
}

func (t *Clone) TestUnsafe() {
	t.Catch(func() {
		type cases struct {
			Name string
			name string
			src  any
			Src  any
		}
		cs := cases{Name: "public name", name: "private name", src: 123, Src: 321}

		dst := clone.Clone(cs)
		t.Require().EqualValues(cs, dst)
	})
}

func (t *Clone) TestBase() {
	t.Catch(func() {

	})
}
