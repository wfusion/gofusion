package http

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"

	fusCtx "github.com/wfusion/gofusion/context"
)

var (
	errParam Errcode = -1
)

type Errcode int

// String Here, i18n cannot be used for localization,
// because in Localize, since the code is of string type, the process of doing toString will lead to stack overflow.
func (e Errcode) String() (r string) {
	return strconv.Itoa(int(e))
}
func (e Errcode) Error() (r string) {
	return I18n.Localize(e)
}

type errOption struct {
	msg   string
	langs []string
	param map[string]any
}

func Langs(c *gin.Context) utils.OptionFunc[errOption] {
	return func(e *errOption) {
		e.langs = langs(c)
	}
}

func Param(param map[string]any) utils.OptionFunc[errOption] {
	return func(e *errOption) {
		e.param = param
	}
}

func Msg(msg string) utils.OptionFunc[errOption] {
	return func(e *errOption) {
		e.msg = msg
	}
}

// Err customized message
func Err(c *gin.Context, code Errcode, opts ...utils.OptionExtender) error {
	opt := utils.ApplyOptions[errOption](opts...)
	if len(opt.langs) == 0 {
		opt.langs = langs(c)
	}
	return &bizErr{
		code:      code,
		errOption: opt,
	}
}

// ErrCtx customized message
func ErrCtx(ctx context.Context, code Errcode, opts ...utils.OptionExtender) error {
	opt := utils.ApplyOptions[errOption](opts...)
	if len(opt.langs) == 0 {
		opt.langs = fusCtx.GetLangs(ctx)
	}
	return &bizErr{
		code:      code,
		errOption: opt,
	}
}

type bizErr struct {
	*errOption
	code Errcode
}

func (b *bizErr) Error() (r string) {
	if b.msg != "" {
		return b.msg
	}
	return I18n.Localize(b.code, i18n.Langs(b.langs), i18n.Param(b.param))
}
