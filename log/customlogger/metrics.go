package customlogger

import (
	"context"
	"reflect"
	"strings"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/infra/metrics"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// MetricsLoggerType FIXME: should not be deleted to avoid compiler optimized
	MetricsLoggerType = reflect.TypeOf(metricsLogger{})
	metricsFields     = log.Fields{"component": strings.ToLower(config.ComponentMetrics)}
)

func DefaultMetricsLogger() metrics.Logger {
	return &metricsLogger{}
}

type metricsLogger struct {
	log      log.Logable
	appName  string
	confName string
	enabled  bool
}

func (m *metricsLogger) Init(log log.Logable, appName, name string) {
	m.log = log
	m.appName = appName
	m.confName = name
	m.reloadConfig()
}

func (m *metricsLogger) Printf(format string, args ...any) {
	if !m.isLoggable() {
		return
	}
	m.logger().Info(context.Background(), format, append(args, metricsFields)...)
}

// Debug logs a message at Debug level.
func (m *metricsLogger) Debug(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	m.logger().Debug(ctx, format, args...)
}

// Info logs a message at Info level.
func (m *metricsLogger) Info(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	m.logger().Info(ctx, format, args...)
}

// Warn logs a message at Warning level.
func (m *metricsLogger) Warn(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	m.logger().Warn(ctx, format, args...)
}

// Error logs a message at Error level.
func (m *metricsLogger) Error(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	m.logger().Error(ctx, format, args...)
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (m *metricsLogger) Fatal(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	m.logger().Fatal(ctx, format, args...)
}

func (m *metricsLogger) logger() log.Logable {
	if m.log != nil {
		return m.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(m.appName))
}

// parseArgs support (ctx, format, args...) log format
func (m *metricsLogger) parseArgs(args ...any) (ctx context.Context, format string, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), "", []any{metricsFields}
	}
	if len(args) == 1 {
		args = append(args, metricsFields)
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

	params = append(params, metricsFields)
	return
}

func (m *metricsLogger) isLoggable() bool {
	if m.confName == "" {
		return true
	}
	m.reloadConfig()
	return m.enabled
}

func (m *metricsLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(m.appName).LoadComponentConfig(config.ComponentMetrics, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[m.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg["enable_logger"])
	m.enabled = enabled
}
