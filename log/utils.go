package log

import (
	"strings"

	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log/encoder"
)

func getLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func getEncoderConfig(conf *Conf) zapcore.EncoderConfig {
	skips := make([]string, 0, len(encoder.SkipCallers)+len(conf.SkipCallers))
	skips = append(skips, encoder.SkipCallers...)
	skips = append(skips, conf.SkipCallers...)
	return zapcore.EncoderConfig{
		LevelKey:       "L",
		TimeKey:        "T",
		MessageKey:     "M",
		NameKey:        "N",
		CallerKey:      "C",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   encoder.SkipCallerEncoder(skips, conf.ShorterFilepath),
	}
}

func getEncoder(layout string, cfg zapcore.EncoderConfig) zapcore.Encoder {
	switch strings.ToLower(layout) {
	case "json":
		return zapcore.NewJSONEncoder(cfg)
	case "console":
		return zapcore.NewConsoleEncoder(cfg)
	default:
		return zapcore.NewJSONEncoder(cfg)
	}
}

func newZapLogLevel(appName, confName, enableField, levelField string) zapcore.LevelEnabler {
	z := &zapLogLevel{
		enabled:     true,
		appName:     config.Use(appName).AppName(),
		confName:    confName,
		enableField: enableField,
		levelField:  levelField,
	}
	z.reloadConfig()
	return z
}

type zapLogLevel struct {
	zapcore.Level

	enabled     bool
	appName     string
	confName    string
	enableField string
	levelField  string
}

func (z *zapLogLevel) Enabled(level zapcore.Level) bool {
	if z.reloadConfig(); !z.enabled {
		return false
	}

	return level >= z.Level
}

func (z *zapLogLevel) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(z.appName).LoadComponentConfig(config.ComponentLog, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[z.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg[z.enableField])
	z.enabled = enabled

	logLevelObj, ok1 := cfg[z.levelField]
	logLevel, ok2 := logLevelObj.(string)
	if !ok1 || !ok2 {
		return
	}
	level := getLogLevel(logLevel)
	z.Level = level
}
