package cases

import (
	"context"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/log/customlogger"

	testCron "github.com/wfusion/gofusion/test/cron"
)

func TestLog(t *testing.T) {
	testingSuite := &Log{Test: new(testCron.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Log struct {
	*testCron.Test
}

func (t *Log) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Log) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Log) TestCustomLogger() {
	t.Catch(func() {
		i := 0
		c := cron.New(
			cron.WithSeconds(),
			cron.WithChain(cron.Recover(cron.DefaultLogger), cron.SkipIfStillRunning(cron.DefaultLogger)),
			cron.WithLocation(constant.DefaultLocation()),
			cron.WithLogger(cron.VerbosePrintfLogger(customlogger.DefaultCronLogger())),
		)
		_, err := c.AddFunc("*/1 * * * * *", func() {
			i++
		})
		t.NoError(err)
		c.Start()
		defer c.Stop()
		time.Sleep(2 * time.Second)
		t.Greater(i, 0)
	})
}
