package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"

	testLog "github.com/wfusion/gofusion/test/log"
)

func TestLog(t *testing.T) {
	testingSuite := &Log{Test: new(testLog.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Log struct {
	*testLog.Test
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

func (t *Log) TestLevel() {
	t.Catch(func() {
		logger := log.Use("default", log.AppName(t.AppName()))

		// When
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Then
		t.EqualValues(log.DebugLevel, logger.Level(ctx))
	})
}

func (t *Log) TestTimeElapsed() {
	t.Catch(func() {
		logger := log.Use("default", log.AppName(t.AppName()))

		// When
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Then
		log.TimeElapsed(ctx, logger, func() {}, "with args %s %v", "1", 2)
		log.TimeElapsed(ctx, logger, func() {}, "without args")
	})
}
