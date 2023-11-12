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
	// CronLoggerType FIXME: should not be deleted to avoid compiler optimized
	CronLoggerType = reflect.TypeOf(cronLogger{})
	cronFields     = log.Fields{"component": strings.ToLower(config.ComponentCron)}
)

func DefaultCronLogger() interface{ Printf(string, ...any) } {
	return &cronLogger{}
}

func DefaultAsynqCronLogger() asynq.Logger {
	return &cronLogger{}
}

type cronLogger struct {
	log      log.Loggable
	appName  string
	confName string
	enabled  bool
}

func (c *cronLogger) Init(log log.Loggable, appName, name string) {
	c.log = log
	c.appName = appName
	c.confName = name
	c.reloadConfig()
}

func (c *cronLogger) Printf(format string, args ...any) {
	if !c.isLoggable() {
		return
	}
	c.logger().Info(context.Background(), format, append(args, cronFields)...)
}

// Debug logs a message at Debug level.
func (c *cronLogger) Debug(args ...any) {
	if !c.isLoggable() {
		return
	}
	ctx, format, args := c.parseArgs(args...)
	c.logger().Debug(ctx, format, args...)
}

// Info logs a message at Info level.
func (c *cronLogger) Info(args ...any) {
	if !c.isLoggable() {
		return
	}
	ctx, format, args := c.parseArgs(args...)
	c.logger().Info(ctx, format, args...)
}

// Warn logs a message at Warning level.
func (c *cronLogger) Warn(args ...any) {
	if !c.isLoggable() {
		return
	}
	ctx, format, args := c.parseArgs(args...)
	c.logger().Warn(ctx, format, args...)
}

// Error logs a message at Error level.
func (c *cronLogger) Error(args ...any) {
	if !c.isLoggable() {
		return
	}
	ctx, format, args := c.parseArgs(args...)
	c.logger().Error(ctx, format, args...)
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (c *cronLogger) Fatal(args ...any) {
	if !c.isLoggable() {
		return
	}
	ctx, format, args := c.parseArgs(args...)
	c.logger().Fatal(ctx, format, args...)
}

func (c *cronLogger) logger() log.Loggable {
	if c.log != nil {
		return c.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(c.appName))
}

// parseArgs support (ctx, format, args...) log format
func (c *cronLogger) parseArgs(args ...any) (ctx context.Context, format string, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), "", []any{cronFields}
	}
	if len(args) == 1 {
		args = append(args, cronFields)
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

	params = append(params, cronFields)
	return
}

func (c *cronLogger) isLoggable() bool {
	if c.confName == "" {
		return true
	}
	c.reloadConfig()
	return c.enabled
}

func (c *cronLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(c.appName).LoadComponentConfig(config.ComponentCron, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[c.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg["enable_logger"])
	c.enabled = enabled
}
