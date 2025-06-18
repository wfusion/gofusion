package http

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constraint"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"

	fusCtx "github.com/wfusion/gofusion/context"
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

func RspError(c *gin.Context, data any, page, count int, msg string, err error, opts ...utils.OptionExtender) {
	var code int
	switch e := err.(type) {
	case Errcode:
		code, msg = int(e), e.Error()
	case *Error:
		code, msg = int(e.Code), e.Error()
	case Error:
		code, msg = int(e.Code), e.Error()
	case *bizErr:
		if e.err != nil {
			code, msg = int(e.err.Code), e.Error()
		} else {
			code, msg = int(e.code), e.Error()
		}
	default:
		code, msg = int(errParam), e.Error()
	}

	r, _ := Use(opts...).(*router)
	rspError(c, r.appName, code, data, page, count, msg)

	go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
		cast.ToInt(code), c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

func RspSuccess(c *gin.Context, data any, page, count int, msg string, opts ...utils.OptionExtender) {
	r, _ := Use(opts...).(*router)
	rspSuccess(c, r.successCode, data, page, count, msg)

	go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
		r.successCode, c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

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
		TraceID: c.GetString(fusCtx.KeyTraceID),
	})
}

func rspError[T constraint.Integer | Error](c *gin.Context, appName string, code T, data any, page, count int, msg string) {
	status := c.Writer.Status()
	if status <= 0 {
		status = http.StatusBadRequest
	}

	if msg == "" {
		switch realCode := any(code).(type) {
		case Errcode:
			msg = Localizable(AppName(appName)).Localize(realCode, i18n.Langs(langs(c)))
		case *Error:
			msg = LocalizableError(AppName(appName)).Localize(*realCode, i18n.Langs(langs(c)))
		case Error:
			msg = LocalizableError(AppName(appName)).Localize(realCode, i18n.Langs(langs(c)))
		default:
			msg = Localizable(AppName(appName)).Localize(Errcode(cast.ToInt(realCode)), i18n.Langs(langs(c)))
		}
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
		TraceID: c.GetString(fusCtx.KeyTraceID),
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
	if lang := c.GetString(fusCtx.KeyLangs); utils.IsStrNotBlank(lang) {
		langs = append(langs, lang)
	}
	return langs
}
