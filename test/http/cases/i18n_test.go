package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/i18n"
	"github.com/wfusion/gofusion/log"

	fusHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestI18n(t *testing.T) {
	testingSuite := &I18n{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type I18n struct {
	*testHtp.Test
}

func (t *I18n) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *I18n) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *I18n) TestCase() {
	t.Catch(func() {
		config.Use(t.AppName()).DI().MustInvoke(func(b i18n.Localizable[fusHtp.Errcode]) {
			b.AddMessages(fusHtp.Errcode(1), map[language.Tag]*i18n.Message{})
		})
	})
}
