package log

import (
	"context"

	"github.com/wfusion/gofusion/common/utils"
)

type Logable interface {
	Debug(ctx context.Context, format string, args ...any)
	Info(ctx context.Context, format string, args ...any)
	Warn(ctx context.Context, format string, args ...any)
	Error(ctx context.Context, format string, args ...any)
	Panic(ctx context.Context, format string, args ...any)
	Fatal(ctx context.Context, format string, args ...any)
	flush()
}

const (
	ErrDuplicatedName        utils.Error = "duplicated logger name"
	ErrUnknownOutput         utils.Error = "unknown logger output"
	ErrDefaultLoggerNotFound utils.Error = "default logger not found"
)

// Conf log configure
//nolint: revive // struct tag too long issue
type Conf struct {
	LogLevel            string `yaml:"log_level" json:"log_level" toml:"log_level" default:"info"`
	StacktraceLevel     string `yaml:"stacktrace_level" json:"stacktrace_level" toml:"stacktrace_level" default:"panic"`
	EnableConsoleOutput bool   `yaml:"enable_console_output" json:"enable_console_output" toml:"enable_console_output" default:"true"`
	ConsoleOutputOption struct {
		Layout   string `yaml:"layout" json:"layout" toml:"layout" default:"console"`
		Colorful bool   `yaml:"colorful" json:"colorful" toml:"colorful" default:"false"`
	} `yaml:"console_output_option" json:"console_output_option" toml:"console_output_option"`
	EnableFileOutput bool `yaml:"enable_file_output" json:"enable_file_output" toml:"enable_file_output" default:"false"`
	FileOutputOption struct {
		Layout        string `yaml:"layout" json:"layout" toml:"layout" default:"json"`
		Path          string `yaml:"path" json:"path" toml:"path"`                               // Log save path
		Name          string `yaml:"name" json:"name" toml:"name"`                               // Name of the saved log, defaults to random generation
		RotationTime  string `yaml:"rotation_time" json:"rotation_time" toml:"rotation_time"`    // Log rotation time interval
		RotationCount int    `yaml:"rotation_count" json:"rotation_count" toml:"rotation_count"` // Maximum number of files to keep
		RotationSize  string `yaml:"rotation_size" json:"rotation_size" toml:"rotation_size"`    // File rotation size
		Compress      bool   `yaml:"compress" json:"compress" toml:"compress" default:"false"`
	} `yaml:"file_output_option" json:"file_output_option" toml:"file_output_option"`
	SkipCallers     []string `yaml:"skip_callers" json:"skip_callers" toml:"skip_callers"`
	ShorterFilepath bool     `yaml:"shorter_filepath" json:"shorter_filepath" toml:"shorter_filepath"`
}