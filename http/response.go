package http

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
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

func rspSuccess(c *gin.Context, code int, data any, page, count int, msg string) {
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

	c.PureJSON(status, &Response{
		Code:    code,
		Message: msg,
		Data:    data,
		Page:    pagePtr,
		Count:   countPtr,
		TraceID: c.GetString(fmkCtx.KeyTraceID),
	})
}

func rspError[T constraint.Integer](c *gin.Context, appName string, code T, data any, page, count int, msg string) {
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
}

func embedResponse(c *gin.Context, data any, err error) {
	status := c.Writer.Status()
	if status == 0 {
		if err != nil {
			status = http.StatusOK
		} else {
			status = http.StatusBadRequest
		}
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
