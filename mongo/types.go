package mongo

import (
	"reflect"

	"github.com/wfusion/gofusion/common/infra/drivers/mongo"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	mgoEvt "go.mongodb.org/mongo-driver/event"
)

const (
	ErrDuplicatedName utils.Error = "duplicated mongo name"
)

var (
	customLoggerType          = reflect.TypeOf((*customLogger)(nil)).Elem()
	customLoggerWithTraceType = reflect.TypeOf((*loggerWithTrace)(nil)).Elem()
)

// Conf
//nolint: revive // struct tag too long issue
type Conf struct {
	mongo.Option          `yaml:",inline" json:",inline" toml:",inline"`
	EnableTrace           bool   `yaml:"enable_trace" json:"enable_trace" toml:"enable_trace" default:"false"`
	TraceProviderInstance string `yaml:"trace_provider_instance" json:"trace_provider_instance" toml:"trace_provider_instance"`
	EnableLogger          bool   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
	LoggerConfig          struct {
		Logger           string   `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.mongoLogger"`
		LogInstance      string   `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
		LoggableCommands []string `yaml:"loggable_commands" json:"loggable_commands" toml:"loggable_commands" default:"[insert,find,update,delete,aggregate,distinct,count,findAndModify]"`
	} `yaml:"logger_config" json:"logger_config" toml:"logger_config"`
}

type customLogger interface {
	logger
	Init(log log.Loggable, appName, name string)
}

type logger interface {
	GetMonitor() *mgoEvt.CommandMonitor
}

type loggerWithTrace interface {
	SetTraceMonitor(traceMonitor *mgoEvt.CommandMonitor)
}
