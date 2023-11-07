package cases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestTime(t *testing.T) {
	t.Parallel()
	testingSuite := &Time{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Time struct {
	*testUtl.Test
}

func (t *Time) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Time) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Time) TestNextJitterIntervalFunc() {
	t.Catch(func() {
		ctx := context.Background()
		t.Run("not symmetric", func() {
			base := time.Second
			ratio := 0.2
			next := utils.NextJitterIntervalFunc(base, 20*base, ratio, 2, false)
			for i := 0; i < 10; i++ {
				interval := next()
				log.Info(ctx, "next not symmetric jitter interval: %s", interval)
				t.Greater(interval, base)
				base *= 2
				if base > 20*time.Second {
					base = 20 * time.Second
				}
			}
		})
		t.Run("symmetric", func() {
			base := time.Second
			ratio := 0.2
			next := utils.NextJitterIntervalFunc(base, 20*base, ratio, 2, true)
			for i := 0; i < 10; i++ {
				interval := next()
				log.Info(ctx, "next symmetric jitter interval: %s", interval)
				t.Greater(interval, base-time.Duration(float64(base)*(ratio/2)))
				base *= 2
				if base > 20*time.Second {
					base = 20 * time.Second
				}
			}
		})
	})
}
