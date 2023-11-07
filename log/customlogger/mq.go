package customlogger

import (
	"context"
	"reflect"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// MQLogger FIXME: should not be deleted to avoid compiler optimized
	MQLogger = reflect.TypeOf(mqLogger{})
)

// mqLogger implements watermill.LoggerAdapter with *zap.Logger.
type mqLogger struct {
	log      log.Logable
	appName  string
	confName string
	enabled  bool
	fields   watermill.LogFields
}

// NewLogger returns new watermill.LoggerAdapter using passed *zap.Logger as backend.
func NewLogger() watermill.LoggerAdapter {
	return new(mqLogger)
}

func (m *mqLogger) Init(log log.Logable, appName, name string) {
	m.log = log
	m.appName = appName
	m.confName = name
	m.reloadConfig()
}

// Error writes error log with message, error and some fields.
func (m *mqLogger) Error(msg string, err error, fields watermill.LogFields) {
	if !m.isLoggable() {
		return
	}
	ctx, fs := m.parseLogFields(fields)
	if err != nil {
		if m.log != nil {
			m.log.Error(ctx, msg+": %s", err, fs)
		} else {
			log.Error(ctx, msg+": %s", err, fs)
		}
	} else {
		if m.log != nil {
			m.log.Error(ctx, msg, err, fs)
		} else {
			log.Error(ctx, msg, err, fs)
		}
	}
}

// Info writes info log with message and some fields.
func (m *mqLogger) Info(msg string, fields watermill.LogFields) {
	if !m.isLoggable() {
		return
	}
	ctx, fs := m.parseLogFields(fields)
	if m.log != nil {
		m.log.Info(ctx, msg, fs)
	} else {
		log.Info(ctx, msg, fs)
	}
}

// Debug writes debug log with message and some fields.
func (m *mqLogger) Debug(msg string, fields watermill.LogFields) {
	if !m.isLoggable() {
		return
	}
	ctx, fs := m.parseLogFields(fields)
	if m.log != nil {
		m.log.Debug(ctx, msg, fs)
	} else {
		log.Debug(ctx, msg, fs)
	}
}

// Trace writes debug log instead of trace log because zap does not support trace level logging.
func (m *mqLogger) Trace(msg string, fields watermill.LogFields) {
	if !m.isLoggable() {
		return
	}
	ctx, fs := m.parseLogFields(fields)
	if m.log != nil {
		m.log.Debug(ctx, msg, fs)
	} else {
		log.Debug(ctx, msg, fs)
	}
}

// With returns new LoggerAdapter with passed fields.
func (m *mqLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &mqLogger{fields: m.fields.Add(fields)}
}

func (m *mqLogger) parseLogFields(fields watermill.LogFields) (ctx context.Context, fs log.Fields) {
	ctx = context.Background()
	fields = m.fields.Add(fields)

	fs = make(log.Fields, len(fields)+1)
	for k, v := range fields {
		if k == watermill.ContextLogFieldKey {
			ctx = v.(context.Context)
			continue
		}
		fs[k] = v
	}
	return
}

func (m *mqLogger) isLoggable() bool { m.reloadConfig(); return m.enabled }

func (m *mqLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(m.appName).LoadComponentConfig(config.ComponentMessageQueue, &cfgs)
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
