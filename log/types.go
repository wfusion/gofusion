package log

import (
	"context"

	"github.com/wfusion/gofusion/common/utils"
	"go.uber.org/zap"
)

type Loggable interface {
	Debug(ctx context.Context, format string, args ...any)
	Info(ctx context.Context, format string, args ...any)
	Warn(ctx context.Context, format string, args ...any)
	Error(ctx context.Context, format string, args ...any)
	Panic(ctx context.Context, format string, args ...any)
	Fatal(ctx context.Context, format string, args ...any)
	Level(ctx context.Context) Level
	Config() *outputConf
	flush()
}

const (
	ErrDuplicatedName        utils.Error = "duplicated logger name"
	ErrUnknownOutput         utils.Error = "unknown logger output"
	ErrDefaultLoggerNotFound utils.Error = "default logger not found"
)

// A Level is a logging priority. Higher levels are more important.
type Level int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// dPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	dPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel

	// InvalidLevel is an invalid value for Level.
	//
	// Core implementations may panic if they see messages of this level.
	InvalidLevel = _maxLevel + 1
)

// Conf log configure
//nolint: revive // struct tag too long issue
type Conf struct {
	LogLevel            string `yaml:"log_level" json:"log_level" toml:"log_level" default:"info"`
	StacktraceLevel     string `yaml:"stacktrace_level" json:"stacktrace_level" toml:"stacktrace_level" default:"panic"`
	EnableConsoleOutput bool   `yaml:"enable_console_output" json:"enable_console_output" toml:"enable_console_output"`
	ConsoleOutputOption struct {
		Layout   string `yaml:"layout" json:"layout" toml:"layout" default:"console"`
		Colorful bool   `yaml:"colorful" json:"colorful" toml:"colorful" default:"false"`
	} `yaml:"console_output_option" json:"console_output_option" toml:"console_output_option"`
	EnableFileOutput bool `yaml:"enable_file_output" json:"enable_file_output" toml:"enable_file_output"`
	FileOutputOption struct {
		Layout         string         `yaml:"layout" json:"layout" toml:"layout" default:"json"`
		Path           string         `yaml:"path" json:"path" toml:"path"` // Log save path
		Name           string         `yaml:"name" json:"name" toml:"name"` // Name of the saved log, defaults to random generation
		RotationMaxAge utils.Duration `yaml:"rotation_max_age" json:"rotation_max_age" toml:"rotation_max_age" default:"30d"`
		RotationCount  int            `yaml:"rotation_count" json:"rotation_count" toml:"rotation_count"`               // Maximum number of files to keep
		RotationSize   string         `yaml:"rotation_size" json:"rotation_size" toml:"rotation_size" default:"100mib"` // File rotation size
		Compress       bool           `yaml:"compress" json:"compress" toml:"compress" default:"false"`
	} `yaml:"file_output_option" json:"file_output_option" toml:"file_output_option"`
	SkipCallers     []string `yaml:"skip_callers" json:"skip_callers" toml:"skip_callers"`
	ShorterFilepath bool     `yaml:"shorter_filepath" json:"shorter_filepath" toml:"shorter_filepath"`
}

type outputConf struct {
	Config    *Conf
	ZapConfig *zap.Config
}
