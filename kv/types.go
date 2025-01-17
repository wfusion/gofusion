package kv

import (
	"context"
	"math/big"
	"reflect"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedName    utils.Error = "duplicated kv name"
	ErrUnsupportedKVType utils.Error = "unsupported kv type"
	ErrNilValue          utils.Error = "nil value"
	ErrInvalidExpiration utils.Error = "invalid expiration"
	ErrKeyAlreadyExists  utils.Error = "key already exists"
	ErrrNotImplement     utils.Error = "not implement"
)

var (
	InvalidVersion = big.NewInt(-1)
)

type Storable interface {
	Get(ctx context.Context, key string, opts ...utils.OptionExtender) Got
	Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Put
	Del(ctx context.Context, key string, opts ...utils.OptionExtender) Del
	Has(ctx context.Context, key string, opts ...utils.OptionExtender) Had

	Paginate(ctx context.Context, pattern string, pageSize int, opts ...utils.OptionExtender) Paginated

	getProxy() any
	close() error
	config() *Conf
}

type Got interface {
	String() string
	Version() Version
	KeyValues() KeyValues
	Err() error
}

type Paginated interface {
	More() bool
	Next() (KeyValues, error)
	SetPageSize(pageSize int)
}

type Put interface {
	LeaseID() string
	Err() error
}

type Del interface {
	Err() error
}

type Had interface {
	Version() Version
	Bool() bool
	Err() error
}

type Version interface {
	Version() *big.Int
}

type option struct {
	expired         time.Duration
	version         int
	leaseID         string
	withPrefix      bool
	withKeysOnly    bool
	withConsistency bool
}

func Prefix() utils.OptionFunc[option] {
	return func(o *option) {
		o.withPrefix = true
	}
}

func KeysOnly() utils.OptionFunc[option] {
	return func(o *option) {
		o.withKeysOnly = true
	}
}

func Expire(expired time.Duration) utils.OptionFunc[option] {
	return func(o *option) {
		o.expired = expired
	}
}

func Ver(v int) utils.OptionFunc[option] {
	return func(o *option) {
		o.version = v
	}
}

func LeaseID(leaseID string) utils.OptionFunc[option] {
	return func(o *option) {
		o.leaseID = leaseID
	}
}

func Consistent() utils.OptionFunc[option] {
	return func(o *option) {
		o.withConsistency = true
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
	Addresses   []string `yaml:"addresses" json:"addresses" toml:"addresses"`
	User        string   `yaml:"user" json:"user" toml:"user"`
	Password    string   `yaml:"password" json:"password" toml:"password" encrypted:""`
	DialTimeout string   `yaml:"dial_timeout" json:"dial_timeout" toml:"dial_timeout" default:"5s"`

	// redis configure
	RedisCluster            bool     `yaml:"redis_cluster" json:"redis_cluster" toml:"redis_cluster"`
	RedisDB                 uint     `yaml:"redis_db" json:"redis_db" toml:"redis_db"`
	RedisHooks              []string `yaml:"redis_hooks" json:"redis_hooks" toml:"redis_hooks" default:"[github.com/wfusion/gofusion/log/customlogger.redisKVLogger]"`
	RedisUnloggableCommands []string `yaml:"redis_unloggable_commands" json:"redis_unloggable_commands" toml:"redis_unloggable_commands" default:"[echo,ping]"`
	RedisReadTimeout        string   `yaml:"redis_read_timeout" json:"redis_read_timeout" toml:"redis_read_timeout" default:"3s"`
	RedisWriteTimeout       string   `yaml:"redis_write_timeout" json:"redis_write_timeout" toml:"redis_write_timeout" default:"3s"`
	RedisMinIdleConns       int      `yaml:"redis_min_idle_conns" json:"redis_min_idle_conns" toml:"redis_min_idle_conns"`
	RedisMaxIdleConns       int      `yaml:"redis_max_idle_conns" json:"redis_max_idle_conns" toml:"redis_max_idle_conns"`
	RedisConnMaxIdleTime    string   `yaml:"redis_conn_max_idle_time" json:"redis_conn_max_idle_time" toml:"redis_conn_max_idle_time" default:"30m"`
	RedisConnMaxLifetime    string   `yaml:"redis_conn_max_life_time" json:"redis_conn_max_life_time" toml:"redis_conn_max_life_time"`
	RedisMaxRetries         int      `yaml:"redis_max_retries" json:"redis_max_retries" toml:"redis_max_retries" default:"3"`
	RedisMinRetryBackoff    string   `yaml:"redis_min_retry_backoff" json:"redis_min_retry_backoff" toml:"redis_min_retry_backoff" default:"8ms"`
	RedisMaxRetryBackoff    string   `yaml:"redis_max_retry_backoff" json:"redis_max_retry_backoff" toml:"redis_max_retry_backoff" default:"512ms"`
	RedisPoolSize           int      `yaml:"redis_pool_size" json:"redis_pool_size" toml:"redis_pool_size"`
	RedisPoolTimeout        string   `yaml:"redis_pool_timeout" json:"redis_pool_timeout" toml:"redis_pool_timeout"`

	// consul configure
	ConsulDatacenter string `yaml:"consul_datacenter" json:"consul_datacenter" toml:"consul_datacenter"`
	ConsulWaitTime   string `yaml:"consul_wait_time" json:"consul_wait_time" toml:"consul_wait_time"`

	// etcd configure
	EtcdAutoSyncInterval     string `yaml:"etcd_auto_sync_interval" json:"etcd_auto_sync_interval" toml:"etcd_auto_sync_interval"`
	EtcdDialKeepAliveTime    string `yaml:"etcd_dial_keep_alive_time" json:"etcd_dial_keep_alive_time" toml:"etcd_dial_keep_alive_time"`
	EtcdDialKeepAliveTimeout string `yaml:"etcd_dial_keep_alive_timeout" json:"etcd_dial_keep_alive_timeout" toml:"etcd_dial_keep_alive_timeout"`
	EtcdRejectOldCluster     bool   `yaml:"etcd_reject_old_cluster" json:"etcd_reject_old_cluster" toml:"etcd_reject_old_cluster"`
	EtcdPermitWithoutStream  bool   `yaml:"etcd_permit_without_stream" json:"etcd_permit_without_stream" toml:"etcd_permit_without_stream"`

	// zookeeper configure
	ZooMaxBufferSize     string `yaml:"zoo_max_buffer_size" json:"zoo_max_buffer_size" toml:"zoo_max_buffer_size" default:"0"`
	ZooMaxConnBufferSize string `yaml:"zoo_max_conn_buffer_size" json:"zoo_max_conn_buffer_size" toml:"zoo_max_conn_buffer_size" default:"1.5mib"`
	ZooLogger            string `yaml:"zoo_logger" json:"zoo_logger" toml:"zoo_logger" default:"github.com/wfusion/gofusion/log/customlogger.zookeeperKVLogger"`
}

type kvType string

const (
	kvTypeRedis  kvType = "redis"
	kvTypeConsul kvType = "consul"
	kvTypeEtcd   kvType = "etcd"
	kvTypeZK     kvType = "zookeeper"
)

type redisCustomLogger interface {
	Init(log log.Loggable, appName, name, logInstance string)
}
