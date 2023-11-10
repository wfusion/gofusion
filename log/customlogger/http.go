package customlogger

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	HttpLoggerType = reflect.TypeOf(httpLogger{})
	httpFields     = log.Fields{"component": strings.ToLower(config.ComponentHttp)}
)

func DefaultHttpLogger() resty.Logger {
	return new(httpLogger)
}

type httpLogger struct {
	log     log.Logable
	appName string
	enabled bool
}

func (h *httpLogger) Init(log log.Logable, appName string) {
	h.log = log
	h.appName = appName
	h.reloadConfig()
}

func (h *httpLogger) Errorf(format string, v ...any) {
	if h.reloadConfig(); h.enabled {
		ctx, args := h.parseArgs(v...)
		h.logger().Error(ctx, format, args...)
	}
}
func (h *httpLogger) Warnf(format string, v ...any) {
	if h.reloadConfig(); h.enabled {
		ctx, args := h.parseArgs(v...)
		h.logger().Info(ctx, format, args...)
	}
}
func (h *httpLogger) Debugf(format string, v ...any) {
	if h.reloadConfig(); h.enabled {
		ctx, args := h.parseArgs(v...)
		h.logger().Debug(ctx, format, args...)
	}
}

func (h *httpLogger) logger() log.Logable {
	if h.log != nil {
		return h.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(h.appName))
}

func (h *httpLogger) parseArgs(args ...any) (ctx context.Context, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), []any{httpFields}
	}
	if len(args) == 1 {
		args = append(args, httpFields)
		return context.Background(), args
	}

	params = args
	ctx, ok = args[0].(context.Context)
	if ok {
		params = args[1:]
	}

	if ctx == nil {
		ctx = context.Background()
	}

	params = append(params, httpFields)
	return
}

func (h *httpLogger) reloadConfig() {
	cfg := make(map[string]any)
	_ = config.Use(h.appName).LoadComponentConfig(config.ComponentHttp, &cfg)

	h.enabled = cast.ToBool(cfg["enable_logger"])
}
