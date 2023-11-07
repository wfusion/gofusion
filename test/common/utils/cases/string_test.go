package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestString(t *testing.T) {
	t.Parallel()
	testingSuite := &String{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type String struct {
	*testUtl.Test
}

func (t *String) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *String) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *String) TestFuzzyKeyword() {
	t.Catch(func() {
		keyword := "user_id"
		fuzzs := utils.FuzzyKeyword(keyword)
		t.Greater(len(fuzzs), 1)
	})
}
