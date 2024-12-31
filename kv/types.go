package kv

import (
	"context"
	"reflect"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedName            utils.Error = "duplicated kv name"
	ErrUnsupportedKVType         utils.Error = "unsupported kv type"
	ErrNilValue                  utils.Error = "nil value"
	ErrUnsupportedRedisValueType utils.Error = "unsupported redis value type"
)

type KeyValue interface {
	Get(ctx context.Context, key string, opts ...utils.OptionExtender) Value
	Set(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Value

	getProxy() any
	close() error
}

type Value interface {
	String() (string, error)
}

type getOption struct {
}

type setOption struct {
	expired time.Duration
}

func Expire(expired time.Duration) utils.OptionFunc[setOption] {
	return func(o *setOption) {
		o.expired = expired
	}
}

var (
	redisCustomLoggerType = reflect.TypeOf((*redisCustomLogger)(nil)).Elem()
)

// Conf
//nolint: revive // struct tag too long issue
type Conf struct {
	Endpoint     *endpointConf `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	Type         kvType        `yaml:"type" json:"type" toml:"type"`
	EnableLogger bool          `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger"`
	LogInstance  string        `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}

type endpointConf struct {
	Addresses []string `yaml:"addresses" json:"addresses" toml:"addresses"`
	User      string   `yaml:"user" json:"user" toml:"user"`
	Password  string   `yaml:"password" json:"password" toml:"password" encrypted:""`

	// redis configure
	Cluster                 bool     `yaml:"cluster" json:"cluster" toml:"cluster"`
	DB                      uint     `yaml:"db" json:"db" toml:"db"`
	DialTimeout             string   `yaml:"dial_timeout" json:"dial_timeout" toml:"dial_timeout" default:"5s"`
	ReadTimeout             string   `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout" default:"3s"`
	WriteTimeout            string   `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout" default:"3s"`
	MinIdleConns            int      `yaml:"min_idle_conns" json:"min_idle_conns" toml:"min_idle_conns"`
	MaxIdleConns            int      `yaml:"max_idle_conns" json:"max_idle_conns" toml:"max_idle_conns"`
	ConnMaxIdleTime         string   `yaml:"conn_max_idle_time" json:"conn_max_idle_time" toml:"conn_max_idle_time" default:"30m"`
	ConnMaxLifetime         string   `yaml:"conn_max_life_time" json:"conn_max_life_time" toml:"conn_max_life_time"`
	MaxRetries              int      `yaml:"max_retries" json:"max_retries" toml:"max_retries" default:"3"`
	MinRetryBackoff         string   `yaml:"min_retry_backoff" json:"min_retry_backoff" toml:"min_retry_backoff" default:"8ms"`
	MaxRetryBackoff         string   `yaml:"max_retry_backoff" json:"max_retry_backoff" toml:"max_retry_backoff" default:"512ms"`
	PoolSize                int      `yaml:"pool_size" json:"pool_size" toml:"pool_size"`
	PoolTimeout             string   `yaml:"pool_timeout" json:"pool_timeout" toml:"pool_timeout"`
	RedisHooks              []string `yaml:"redis_hooks" json:"redis_hooks" toml:"redis_hooks" default:"[github.com/wfusion/gofusion/log/customlogger.redisKVLogger]"`
	RedisUnloggableCommands []string `yaml:"redis_unloggable_commands" json:"redis_unloggable_commands" toml:"redis_unloggable_commands" default:"[echo,ping]"`
}

type kvType string

const (
	kvTypeRedis  kvType = "redis"
	kvTypeConsul kvType = "consul"
	kvTypeEtcd   kvType = "etcd"
	kvTypeZK     kvType = "zookeeper"
	kvTypeEureka kvType = "eureka"
)

type redisCustomLogger interface {
	Init(log log.Loggable, appName, name string)
}
