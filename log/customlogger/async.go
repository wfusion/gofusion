package customlogger

import (
	"context"
	"reflect"
	"strings"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// AsyncLoggerType FIXME: should not be deleted to avoid compiler optimized
	AsyncLoggerType = reflect.TypeOf(asyncLogger{})
	asyncFields     = log.Fields{"component": strings.ToLower(config.ComponentAsync)}
)

func DefaultAsyncLogger() interface{ Printf(string, ...any) } {
	return &asyncLogger{}
}

func DefaultAsynqAsyncLogger() asynq.Logger {
	return &asyncLogger{}
}

type asyncLogger struct {
	log      log.Loggable
	appName  string
	confName string
	enabled  bool
}

func (a *asyncLogger) Init(log log.Loggable, appName, confName string) {
	a.log = log
	a.appName = appName
	a.confName = confName
	a.reloadConfig()
}

func (a *asyncLogger) Printf(format string, args ...any) {
	if !a.isLoggable() {
		return
	}
	a.logger().Info(context.Background(), format, append(args, asyncFields)...)
}

// Debug logs a message at Debug level.
func (a *asyncLogger) Debug(args ...any) {
	if !a.isLoggable() {
		return
	}
	ctx, format, args := a.parseArgs(args...)
	a.logger().Debug(ctx, format, args...)
}

// Info logs a message at Info level.
func (a *asyncLogger) Info(args ...any) {
	if !a.isLoggable() {
		return
	}
	ctx, format, args := a.parseArgs(args...)
	a.logger().Info(ctx, format, args...)
}

// Warn logs a message at Warning level.
func (a *asyncLogger) Warn(args ...any) {
	if !a.isLoggable() {
		return
	}
	ctx, format, args := a.parseArgs(args...)
	a.logger().Warn(ctx, format, args...)
}

// Error logs a message at Error level.
func (a *asyncLogger) Error(args ...any) {
	if !a.isLoggable() {
		return
	}
	ctx, format, args := a.parseArgs(args...)
	a.logger().Error(ctx, format, args...)
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (a *asyncLogger) Fatal(args ...any) {
	if !a.isLoggable() {
		return
	}
	ctx, format, args := a.parseArgs(args...)
	a.logger().Fatal(ctx, format, args...)
}

func (a *asyncLogger) logger() log.Loggable {
	if a.log != nil {
		return a.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(a.appName))
}

// parseArgs support (ctx, format, args...) log format
func (a *asyncLogger) parseArgs(args ...any) (ctx context.Context, format string, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), "", []any{asyncFields}
	}
	if len(args) == 1 {
		args = append(args, asyncFields)
		return context.Background(), "%+v", args
	}

	format, ok = args[0].(string)
	if ok {
		params = args[1:]
	} else {
		ctx, _ = args[0].(context.Context)
		format, _ = args[1].(string)
		params = args[2:]
	}
	if format == "" {
		placeholder := make([]string, len(args))
		for i := 0; i < len(args); i++ {
			placeholder[i] = "%+v"
		}
		format = strings.Join(placeholder, " ")
		params = args
	}

	if ctx == nil {
		ctx = context.Background()
	}

	params = append(params, asyncFields)
	return
}

func (a *asyncLogger) isLoggable() bool {
	if a.confName == "" {
		return true
	}
	a.reloadConfig()
	return a.enabled
}

func (a *asyncLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(a.appName).LoadComponentConfig(config.ComponentAsync, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[a.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg["enable_logger"])
	a.enabled = enabled
}
