package log

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
