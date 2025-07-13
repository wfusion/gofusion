package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/config"
)

func TestRemote(t *testing.T) {
	testingSuite := &Remote{Test: new(config.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Remote struct {
	*config.Test
}

func (t *Remote) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Remote) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}
