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
	"github.com/wfusion/gofusion/log"

	fmkHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestGroup(t *testing.T) {
	testingSuite := &Group{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Group struct {
	*testHtp.Test
}

func (t *Group) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Group) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Group) TestGroupDispatch() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id" binding:"required"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		method := http.MethodPost
		path := "/test"
		group := "/group"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return io.ErrUnexpectedEOF
		}
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&reqStruct{
			ID:      utils.AnyPtr(utils.UUID()),
			NumList: []int{1, 2, 3, 4, 5, 6},
			Embed: &struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			}{
				FUID:    utils.AnyPtr(utils.UUID()),
				Boolean: utils.AnyPtr(true),
			},
		}))

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, group+path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)

		groupRouter := fmkHtp.Use(fmkHtp.AppName(t.AppName())).Group(group)
		groupRouter.POST(path, hd)

		// When
		fmkHtp.Use(fmkHtp.AppName(t.AppName())).ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}
