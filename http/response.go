package http

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constraint"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"

	fmkCtx "github.com/wfusion/gofusion/context"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`

	// pagination
	Page  *int `json:"page,omitempty"`
	Count *int `json:"count,omitempty"`

	// Trace
	TraceID string `json:"traceid"`
}

type Embed struct {
}

var (
	embedType    = reflect.TypeOf(Embed{})
	responseType = reflect.TypeOf(Response{})
)

func Success(c *gin.Context, appName string, data any, page, count int, msg string) {
	status := c.Writer.Status()
	if status <= 0 {
		status = http.StatusOK
	}
	if msg == "" {
		msg = "ok"
	}
	var (
		pagePtr  *int
		countPtr *int
	)
	if page > 0 {
		pagePtr = utils.AnyPtr(page)
	}
	if count >= 0 {
		countPtr = utils.AnyPtr(count)
	}

	code := Use(AppName(appName)).Config().SuccessCode
	c.PureJSON(status, &Response{
		Code:    code,
		Message: msg,
		Data:    data,
		Page:    pagePtr,
		Count:   countPtr,
		TraceID: c.GetString(fmkCtx.KeyTraceID),
	})

	go metricsCode(fmkCtx.New(fmkCtx.Gin(c)), appName, c.Request.URL.Path, c.Request.Method, code,
		c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

func Error[T constraint.Integer](c *gin.Context, appName string, code T, data any, page, count int, msg string) {
	status := c.Writer.Status()
	if status <= 0 {
		status = http.StatusBadRequest
	}

	if msg == "" {
		msg = Localizable(AppName(appName)).Localize(Errcode(code), i18n.Langs(langs(c)))
	}
	var (
		pagePtr  *int
		countPtr *int
	)
	if page > 0 {
		pagePtr = utils.AnyPtr(page)
	}
	if count >= 0 {
		countPtr = utils.AnyPtr(count)
	}

	c.PureJSON(status, &Response{
		Code:    cast.ToInt(code),
		Message: msg,
		Data:    data,
		Page:    pagePtr,
		Count:   countPtr,
		TraceID: c.GetString(fmkCtx.KeyTraceID),
	})

	go metricsCode(fmkCtx.New(fmkCtx.Gin(c)), appName, c.Request.URL.Path, c.Request.Method, cast.ToInt(code),
		c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

func embedResponse(c *gin.Context, appName string, data any, err error) {
	status := c.Writer.Status()
	if status == 0 {
		if err != nil {
			status = http.StatusOK
		} else {
			status = http.StatusBadRequest
		}
	}

	ctx := fmkCtx.New(fmkCtx.Gin(c))
	switch rsp := data.(type) {
	case Response:
		go metricsCode(ctx, appName, c.Request.URL.Path, c.Request.Method, rsp.Code,
			c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	case *Response:
		go metricsCode(ctx, appName, c.Request.URL.Path, c.Request.Method, rsp.Code,
			c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	default:
		rspData := make(map[string]any)
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &rspData,
			TagName: "json",
		})
		if err == nil && dec != nil {
			_ = dec.Decode(data)
		}
		var code any
		utils.IfAny(
			func() (ok bool) { code, ok = rspData["code"]; return ok },
			func() (ok bool) { code, ok = rspData["Code"]; return ok },
		)
		if code == nil {
			code = -2
		}
		go metricsCode(ctx, appName, c.Request.URL.Path, c.Request.Method, cast.ToInt(code),
			c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	}

	c.PureJSON(status, data)
}

func langs(c *gin.Context) (langs []string) {
	if c == nil {
		return
	}
	langs = c.Request.Header.Values("Accept-Language")
	if lang := c.GetString("lang"); utils.IsStrNotBlank(lang) {
		langs = append(langs, lang)
	}
	if lang := c.GetString(fmkCtx.KeyLangs); utils.IsStrNotBlank(lang) {
		langs = append(langs, lang)
	}
	return langs
}
