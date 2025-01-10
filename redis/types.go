package redis

import (
	"reflect"

	"github.com/wfusion/gofusion/common/infra/drivers/redis"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedName utils.Error = "duplicated redis name"
)

var (
	customLoggerType = reflect.TypeOf((*customLogger)(nil)).Elem()
)

// Conf
//nolint: revive // struct tag too long issue
type Conf struct {
	redis.Option       `yaml:",inline" json:",inline" toml:",inline"`
	Hooks              []string `yaml:"hooks" json:"hooks" toml:"hooks" default:"[github.com/wfusion/gofusion/log/customlogger.redisLogger]"`
	EnableLogger       bool     `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger"`
	LogInstance        string   `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
	UnloggableCommands []string `yaml:"unloggable_commands" json:"unloggable_commands" toml:"unloggable_commands" default:"[echo,ping]"`
}

type customLogger interface {
	Init(log log.Loggable, appName, name, logInstance string)
}
