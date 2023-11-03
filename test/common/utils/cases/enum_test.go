package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestEnum(t *testing.T) {
	testingSuite := &Enum{Test: testUtl.T}
	suite.Run(t, testingSuite)
}

type Enum struct {
	*testUtl.Test
}

func (t *Enum) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Enum) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Enum) TestSoftDeleteStatus() {
	t.Catch(func() {
		ctx := context.Background()
		unknown := compress.AlgorithmUnknown
		log.Info(ctx, "%s", unknown)
	})
}
