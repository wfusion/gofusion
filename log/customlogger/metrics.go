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
	m.log.Info(context.Background(), format, args...)
}

// Debug logs a message at Debug level.
func (m *metricsLogger) Debug(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	if m.log != nil {
		m.log.Debug(ctx, format, args...)
	} else {
		log.Debug(ctx, format, args...)
	}
}

// Info logs a message at Info level.
func (m *metricsLogger) Info(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	if m.log != nil {
		m.log.Info(ctx, format, args...)
	} else {
		log.Info(ctx, format, args...)
	}
}

// Warn logs a message at Warning level.
func (m *metricsLogger) Warn(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	if m.log != nil {
		m.log.Warn(ctx, format, args...)
	} else {
		log.Warn(ctx, format, args...)
	}
}

// Error logs a message at Error level.
func (m *metricsLogger) Error(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	if m.log != nil {
		m.log.Error(ctx, format, args...)
	} else {
		log.Error(ctx, format, args...)
	}
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (m *metricsLogger) Fatal(args ...any) {
	if !m.isLoggable() {
		return
	}
	ctx, format, args := m.parseArgs(args...)
	if m.log != nil {
		m.log.Fatal(ctx, format, args...)
	} else {
		log.Fatal(ctx, format, args...)
	}
}

// parseArgs support (ctx, format, args...) log format
func (m *metricsLogger) parseArgs(args ...any) (ctx context.Context, format string, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), "", nil
	}
	if len(args) == 1 {
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

	return
}

func (m *metricsLogger) isLoggable() bool { m.reloadConfig(); return m.enabled }

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
