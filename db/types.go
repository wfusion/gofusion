package db

import (
	"reflect"

	"gorm.io/gorm/clause"

	"github.com/wfusion/gofusion/common/infra/drivers/orm"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedName   utils.Error = "duplicated database name"
	ErrDatabaseNotFound utils.Error = "not found database to use"
)

var (
	customLoggerType         = reflect.TypeOf((*customLogger)(nil)).Elem()
	gormClauseExpressionType = reflect.TypeOf((*clause.Expression)(nil)).Elem()
)

// Conf
//nolint: revive // struct tag too long issue
type Conf struct {
	orm.Option             `yaml:",inline" json:",inline" toml:",inline"`
	AutoIncrementIncrement int64          `yaml:"auto_increment_increment" json:"auto_increment_increment" toml:"auto_increment_increment"`
	Sharding               []shardingConf `yaml:"sharding" json:"sharding" toml:"sharding"`
	EnableTrace            bool           `yaml:"enable_trace" json:"enable_trace" toml:"enable_trace"`
	TraceProviderInstance  string         `yaml:"trace_provider_instance" json:"trace_provider_instance" toml:"trace_provider_instance"`
	EnableLogger           bool           `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger"`
	LoggerConfig           struct {
		Logger        string `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.gormLogger"`
		LogInstance   string `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
		LogLevel      string `yaml:"log_level" json:"log_level" toml:"log_level"`
		SlowThreshold string `yaml:"slow_threshold" json:"slow_threshold" toml:"slow_threshold"`
	} `yaml:"logger_config" json:"logger_config" toml:"logger_config"`
}

// shardingConf
//nolint: revive // struct tag too long issue
type shardingConf struct {
	Table                    string   `yaml:"table"`
	Suffix                   string   `yaml:"suffix"`
	Columns                  []string `yaml:"columns"`
	ShardingKeyExpr          string   `yaml:"sharding_key_expr"`
	ShardingKeyByRawValue    bool     `yaml:"sharding_key_by_raw_value"`
	ShardingKeysForMigrating []string `yaml:"sharding_keys_for_migrating"`
	NumberOfShards           uint     `yaml:"number_of_shards"`
	IDGen                    string   `yaml:"idgen" default:"github.com/wfusion/gofusion/common/infra/drivers/orm/idgen.NewSnowflake"`
}

type customLogger interface {
	Init(log log.Loggable, appName, name string)
}
