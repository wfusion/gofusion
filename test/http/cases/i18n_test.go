package cases

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/common/utils"
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

func (t *I18n) TestI18n() {
	t.Catch(func() {
		config.Use(t.AppName()).DI().MustInvoke(func(b i18n.Localizable[fusHtp.Errcode]) {
			b.AddMessages(fusHtp.Errcode(1), map[language.Tag]*i18n.Message{})
		})
	})
}

func (t *I18n) TestValidator() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id" binding:"required,max=256"`
			NumList []int   `json:"num_list" binding:"required,len=2"`
			FUID    *string `json:"fuid" binding:"required,uuid"`
			Email   *string `json:"email" binding:"required,email"`
			Gender  *string `json:"gender" binding:"required,oneof=female male"`
			Embed   *struct {
				Boolean *bool `json:"boolean" binding:"required"`
			} `json:"embed" binding:"required"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return io.ErrUnexpectedEOF
		}
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&reqStruct{
			ID:      utils.AnyPtr(utils.UUID()),
			NumList: []int{1, 2},
			FUID:    utils.AnyPtr(utils.UUID()),
			Email:   utils.AnyPtr("gofusion_mail.com"),
			Gender:  utils.AnyPtr("female"),
			Embed: &struct {
				Boolean *bool `json:"boolean" binding:"required"`
			}{
				Boolean: utils.AnyPtr(true),
			},
		}))

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		rsp := utils.MustJsonUnmarshal[fusHtp.Response](w.Body.Bytes())
		t.Equal(rsp.Code, -1)
	})
}
