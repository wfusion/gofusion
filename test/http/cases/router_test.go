package cases

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	fmkHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestRouter(t *testing.T) {
	testingSuite := &Router{Test: testHtp.T}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Router struct {
	*testHtp.Test
}

func (t *Router) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Router) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

// case: func(c *gin.Context)
func (t *Router) TestExample01() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		rspBody := map[string]any{"code": 0, "msg": "ok"}
		hd := func(c *gin.Context) {
			c.JSON(http.StatusOK, rspBody)
		}

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, nil)
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Equal(string(utils.MustJsonMarshal(rspBody)), w.Body.String())
	})
}

// case: func(c *gin.Context, req *Struct FromJsonBody) error
func (t *Router) TestExample02() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *Struct FromQuery) error
func (t *Router) TestExample03() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
		}

		method := http.MethodGet
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			return io.ErrUnexpectedEOF
		}
		uri := path + "?id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}"

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, uri, nil)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (rsp *Struct{Data *struct; Page, Count int; Msg string}, err error)
func (t *Router) TestExample04() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type rspStruct struct {
			Page  int
			Count int
			Msg   string
			Data  *struct {
				Name *string `json:"name"`
				UUID *string `json:"uuid"`
			}
		}

		method := http.MethodPost
		path := "/test"
		data := &struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*rspStruct, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return &rspStruct{Page: 1, Count: 1, Msg: "ok", Data: data}, nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())
		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (rsp *Struct{data *struct; page, count int; msg string}, err error)
func (t *Router) TestExample05() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type rspStruct struct {
			page  int
			count int
			msg   string
			data  *struct {
				Name *string `json:"name"`
				UUID *string `json:"uuid"`
			}
		}

		method := http.MethodPost
		path := "/test"
		data := &struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*rspStruct, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return &rspStruct{page: 1, count: 1, msg: "ok", data: data}, nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (rsp *Struct{data struct; page, count int; msg string}, err error)
func (t *Router) TestExample06() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type rspStruct struct {
			page  int
			count int
			msg   string
			data  struct {
				Name *string `json:"name"`
				UUID *string `json:"uuid"`
			}
		}

		method := http.MethodPost
		path := "/test"
		data := &struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*rspStruct, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return &rspStruct{page: 1, count: 1, msg: "ok", data: *data}, nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (rsp map[string]any, err error)
func (t *Router) TestExample07() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		expRsp := map[string]any{
			"page":  1,
			"count": 1,
			"msg":   "ok",
			"data": map[string]any{
				"name": utils.AnyPtr("what's your name"),
				"uuid": utils.AnyPtr(utils.UUID()),
			},
		}
		hd := func(c *gin.Context, req *reqStruct) (map[string]any, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return expRsp, nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(expRsp["data"]), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (data *struct, page, count int, msg string, err error)
func (t *Router) TestExample08() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type dataStruct struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}

		method := http.MethodPost
		path := "/test"
		data := &dataStruct{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*dataStruct, int, int, string, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return data, 1, 1, "ok", nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())
		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (data map[string]any, page, count int, err error)
func (t *Router) TestExample09() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		data := map[string]any{
			"name": utils.AnyPtr("what's your name"),
			"uuid": utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (map[string]any, int, int, string, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return data, 1, 1, "ok", nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (data struct, page, count *int64, msg *string, err error)
func (t *Router) TestExample10() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type dataStruct struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}

		method := http.MethodPost
		path := "/test"
		data := &dataStruct{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (dataStruct, *int64, *int64, *string, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return *data, utils.AnyPtr(int64(1)), utils.AnyPtr(int64(1)), utils.AnyPtr("ok"), nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (data []map[string]any, page, count int, err error)
func (t *Router) TestExample11() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		data := []map[string]any{
			{
				"name": utils.AnyPtr("what's your name"),
				"uuid": utils.AnyPtr(utils.UUID()),
			},
		}
		hd := func(c *gin.Context, req *reqStruct) ([]map[string]any, int, int, string, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return data, 1, 1, "ok", nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())

		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("ok", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *Struct FromFormUrlDecodedBody) error
func (t *Router) TestExample12() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			return io.ErrUnexpectedEOF
		}
		body := strings.NewReader("id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}")
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *Struct FromMultipartFormDataBody) error
func (t *Router) TestExample13() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
			File    []byte  `json:"file"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.File)
			return io.ErrUnexpectedEOF
		}

		reqBody := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(reqBody)

		// write file
		filePath := fmt.Sprintf("%s/configs/%s.app.yml", env.WorkDir, testHtp.Component)
		part, err := writer.CreateFormFile("file", filePath)
		t.Require().NoError(err)
		file, err := os.Open(filePath)
		t.Require().NoError(err)
		_, err = io.Copy(part, file)
		t.Require().NoError(err)

		// write field
		reqMap := map[string]string{
			"id":       utils.UUID(),
			"num_list": string(utils.MustJsonMarshal([]int{1, 2, 3, 4, 5, 6})),
			"embed": string(utils.MustJsonMarshal(&struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			}{
				FUID:    utils.AnyPtr(utils.UUID()),
				Boolean: utils.AnyPtr(true),
			})),
		}
		for k, v := range reqMap {
			t.Require().NoError(writer.WriteField(k, v))
		}
		utils.CloseAnyway(writer)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: object.public.method(c *gin.Context, req *FromJsonBody) error
func (t *Router) TestExample14() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&routerReqStruct{
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, t.handle)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: object.private.method(c *gin.Context, req *FromJsonBody) error
func (t *Router) TestExample15() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&routerReqStruct{
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, t.handle)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *FromJsonBody) error, http redirect
func (t *Router) TestExample16() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)

			c.Redirect(http.StatusTemporaryRedirect, "https://ctyun.cn")
			return nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusTemporaryRedirect, w.Code)
		t.Contains(w.Header().Get("Location"), "ctyun")
	})
}

// case: func(c *gin.Context, req map[string]any FromJsonBody) error
func (t *Router) TestExample17() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req map[string]any) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req["id"])
			t.Require().NotEmpty(req["num_list"])
			t.Require().NotEmpty(req["embed"])
			t.Require().NotEmpty(req["embed"].(map[string]any)["fuid"])
			t.Require().NotNil(req["embed"].(map[string]any)["boolean"])
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req []map[string]any FromJsonBody) error
func (t *Router) TestExample18() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req []map[string]any) error {
			t.Require().NotEmpty(req)
			req1 := req[0]
			t.Require().NotNil(req1)
			t.Require().NotEmpty(req1["id"])
			t.Require().NotEmpty(req1["num_list"])
			t.Require().NotEmpty(req1["embed"])
			t.Require().NotEmpty(req1["embed"].(map[string]any)["fuid"])
			t.Require().NotNil(req1["embed"].(map[string]any)["boolean"])
			return io.ErrUnexpectedEOF
		}
		reqBody := bytes.NewReader(utils.MustJsonMarshal([]any{
			&reqStruct{
				ID:      utils.AnyPtr(utils.UUID()),
				NumList: []int{1, 2, 3, 4, 5, 6},
				Embed: &struct {
					FUID    *string `json:"fuid"`
					Boolean *bool   `json:"boolean"`
				}{
					FUID:    utils.AnyPtr(utils.UUID()),
					Boolean: utils.AnyPtr(true),
				},
			},
			struct {
				ID uint `json:"id"`
			}{
				ID: 1,
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
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req Struct FromJsonBody) error
func (t *Router) TestExample19() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req reqStruct) error {
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req map[string]any FromFormUrlDecodedBody) error
func (t *Router) TestExample20() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req map[string]any) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req["id"])
			t.Require().NotEmpty(req["num_list"])
			t.Require().NotEmpty(req["embed"])
			return io.ErrUnexpectedEOF
		}
		body := strings.NewReader("id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}")
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req []map[string]any FromFormUrlDecodedBody) error
func (t *Router) TestExample21() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req []map[string]any) error {
			t.Require().NotEmpty(req)
			req1 := req[0]
			t.Require().NotNil(req1)
			t.Require().NotEmpty(req1["id"])
			t.Require().NotEmpty(req1["num_list"])
			t.Require().NotEmpty(req1["embed"])
			return io.ErrUnexpectedEOF
		}
		body := strings.NewReader("id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}")
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req Struct FromFormUrlDecodedBody) error
func (t *Router) TestExample22() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			return io.ErrUnexpectedEOF
		}
		body := strings.NewReader("id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}")
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req map[string]any FromQuery) error
func (t *Router) TestExample23() {
	t.Catch(func() {
		// Given
		method := http.MethodGet
		path := "/test"
		hd := func(c *gin.Context, req map[string]any) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req["id"])
			t.Require().NotEmpty(req["num_list"])
			t.Require().NotEmpty(req["embed"])
			return io.ErrUnexpectedEOF
		}
		uri := path + "?id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}"

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, uri, nil)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req []map[string]any FromQuery) error
func (t *Router) TestExample24() {
	t.Catch(func() {
		// Given
		method := http.MethodGet
		path := "/test"
		hd := func(c *gin.Context, req []map[string]any) error {
			t.Require().NotEmpty(req)
			req1 := req[0]
			t.Require().NotNil(req1)
			t.Require().NotEmpty(req1["id"])
			t.Require().NotEmpty(req1["num_list"])
			t.Require().NotEmpty(req1["embed"])
			return io.ErrUnexpectedEOF
		}
		uri := path + "?id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}"

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, uri, nil)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req Struct FromQuery) error
func (t *Router) TestExample25() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
		}

		method := http.MethodGet
		path := "/test"
		hd := func(c *gin.Context, req reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			return io.ErrUnexpectedEOF
		}
		uri := path + "?id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}"

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, uri, nil)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req Struct FromMultipartFormDataBody) error
func (t *Router) TestExample28() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
			File    []byte  `json:"file"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.File)
			return io.ErrUnexpectedEOF
		}

		reqBody := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(reqBody)

		// write file
		filePath := fmt.Sprintf("%s/configs/%s.app.yml", env.WorkDir, testHtp.Component)
		part, err := writer.CreateFormFile("file", filePath)
		t.Require().NoError(err)
		file, err := os.Open(filePath)
		t.Require().NoError(err)
		_, err = io.Copy(part, file)
		t.Require().NoError(err)

		// write field
		reqMap := map[string]string{
			"id":       utils.UUID(),
			"num_list": string(utils.MustJsonMarshal([]int{1, 2, 3, 4, 5, 6})),
			"embed": string(utils.MustJsonMarshal(&struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			}{
				FUID:    utils.AnyPtr(utils.UUID()),
				Boolean: utils.AnyPtr(true),
			})),
		}
		for k, v := range reqMap {
			t.Require().NoError(writer.WriteField(k, v))
		}
		utils.CloseAnyway(writer)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (rsp *Struct{Embed}, err error)
func (t *Router) TestExample29() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type rspStruct struct {
			fmkHtp.Embed

			Page  int
			Count int
			Msg   string
			Data  *struct {
				Name *string `json:"name"`
				UUID *string `json:"uuid"`
			}
		}

		method := http.MethodPost
		path := "/test"
		data := &struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*rspStruct, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return &rspStruct{Page: 1, Count: 1, Msg: "ok", Data: data}, nil
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[rspStruct](w.Body.Bytes())
		t.Require().EqualValues(1, rsp.Page)
		t.Require().EqualValues(1, rsp.Count)
		t.Require().EqualValues("ok", rsp.Msg)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *Struct FromQuery) error
func (t *Router) TestExample30() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *string `json:"embed"`
		}

		method := http.MethodGet
		path := "/test/:id"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			return io.ErrUnexpectedEOF
		}
		uri := "/test/b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"?num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}"

		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, uri, nil)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *FromJsonBody) (data *struct , page, count int, msg string, err error) deal error
func (t *Router) TestExample31() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}
		type dataStruct struct {
			Name *string `json:"name"`
			UUID *string `json:"uuid"`
		}

		method := http.MethodPost
		path := "/test"
		data := &dataStruct{
			Name: utils.AnyPtr("what's your name"),
			UUID: utils.AnyPtr(utils.UUID()),
		}
		hd := func(c *gin.Context, req *reqStruct) (*dataStruct, int, int, string, error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return data, 1, 1, "", errors.New("wrong")
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)

		rsp := utils.MustJsonUnmarshal[fmkHtp.Response](w.Body.Bytes())
		t.Require().EqualValues(1, *rsp.Page)
		t.Require().EqualValues(1, *rsp.Count)
		t.Require().EqualValues("wrong", rsp.Message)
		t.Require().NotEmpty(rsp.Data)
		t.Require().EqualValues(utils.MustJsonMarshal(data), utils.MustJsonMarshal(rsp.Data))
	})
}

// case: func(c *gin.Context, req *Struct FromJsonBody) (data []int, err error)
func (t *Router) TestExample32() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) (data []int, err error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return []int{1, 2, 3, 4, 5}, io.ErrUnexpectedEOF
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
		t.Contains(w.Body.String(), "[1,2,3,4,5]")
	})
}

// case: func(c *gin.Context, req *Struct FromJsonBody) (data map[string]int, err error)
func (t *Router) TestExample33() {
	t.Catch(func() {
		// Given
		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *routerReqStruct) (data map[string]int, err error) {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			return map[string]int{"1": 1, "2": 2, "3": 3}, io.ErrUnexpectedEOF
		}
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&routerReqStruct{
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
		t.Contains(w.Body.String(), `"3":3`)
	})
}

// case: func(c *gin.Context, req Struct FromFormUrlDecodedBody) error
func (t *Router) TestExample34() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID       *string `json:"id" form:"id"`
			NumList  []int   `json:"num_list" form:"num_list"`
			Embed    *string `json:"embed" form:"embed"`
			PageSize int     `json:"pageSize" form:"pageSize,default=10"`
			PageNo   int     `json:"pageNo" form:"pageNo,default=1"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotZero(req.PageNo)
			t.Require().NotZero(req.PageSize)
			return io.ErrUnexpectedEOF
		}
		body := strings.NewReader("id=b5890985-47e1-4eca-9dc8-ec95060e896d" +
			"&num_list=1&num_list=2&num_list=3&num_list=4&num_list=5&num_list=6" +
			"&embed={\\\"fuid\\\":\\\"b5890985-47e1-4eca-9dc8-ec95060e896d\\\",\\\"boolean\\\":true}}")
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

// case: func(c *gin.Context, req *Struct FromJsonBody) error with default tag
func (t *Router) TestExample35() {
	t.Catch(func() {
		// Given
		type reqStruct struct {
			ID      *string `json:"id" default:"this is a id"`
			NumList []int   `json:"num_list"`
			Embed   *struct {
				FUID    *string `json:"fuid"`
				Boolean *bool   `json:"boolean"`
			} `json:"embed"`
			M map[string]any `json:"m" default:"a: aaa\nb: 123\nc: {cc: 112233}"`
			S []any          `json:"s" default:"[{a: aaa, b: 123, c: {cc: 112233}}, ok, 123, d: 666]"`
		}

		method := http.MethodPost
		path := "/test"
		hd := func(c *gin.Context, req *reqStruct) error {
			t.Require().NotNil(req)
			t.Require().NotEmpty(req.ID)
			t.Require().NotEmpty(req.NumList)
			t.Require().NotEmpty(req.Embed)
			t.Require().NotEmpty(req.Embed.FUID)
			t.Require().NotNil(req.Embed.Boolean)
			t.Require().NotEmpty(req.M)
			t.Require().NotEmpty(req.S)
			return io.ErrUnexpectedEOF
		}
		reqBody := bytes.NewReader(utils.MustJsonMarshal(&reqStruct{
			ID:      nil,
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
		req, err := http.NewRequest(method, path, reqBody)
		req.Header.Set("Content-Type", "application/json")
		t.Require().NoError(err)
		engine := t.ServerGiven(method, path, hd)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.Contains(w.Body.String(), io.ErrUnexpectedEOF.Error())
	})
}

type routerReqStruct struct {
	ID      *string `json:"id"`
	NumList []int   `json:"num_list"`
	Embed   *struct {
		FUID    *string `json:"fuid"`
		Boolean *bool   `json:"boolean"`
	} `json:"embed"`
}

func (t *Router) handle(c *gin.Context, req *routerReqStruct) error {
	t.Require().NotNil(req)
	t.Require().NotEmpty(req.ID)
	t.Require().NotEmpty(req.NumList)
	t.Require().NotEmpty(req.Embed)
	t.Require().NotEmpty(req.Embed.FUID)
	t.Require().NotNil(req.Embed.Boolean)
	return io.ErrUnexpectedEOF
}
