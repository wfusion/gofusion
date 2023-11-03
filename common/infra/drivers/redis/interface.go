package redis

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/wfusion/gofusion/common/utils"
)

type Dialect interface {
	New(ctx context.Context, option Option, opts ...utils.OptionExtender) (redis *Redis, err error)
}

type newOption struct {
	hooks []redis.Hook
}

type Option struct {
	Cluster   bool     `yaml:"cluster" json:"cluster" toml:"cluster"`
	Endpoints []string `yaml:"endpoints" json:"endpoints" toml:"endpoints"`
	DB        uint     `yaml:"db" json:"db" toml:"db"`
	User      string   `yaml:"user" json:"user" toml:"user"`
	Password  string   `yaml:"password" json:"password" toml:"password" encrypted:""`

	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout string `yaml:"dial_timeout" json:"dial_timeout" toml:"dial_timeout" default:"5s"`
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Supported values:
	//   - `0` - default timeout (3 seconds).
	//   - `-1` - no timeout (block indefinitely).
	//   - `-2` - disables SetReadDeadline calls completely.
	ReadTimeout string `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout" default:"3s"`
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.  Supported values:
	//   - `0` - default timeout (3 seconds).
	//   - `-1` - no timeout (block indefinitely).
	//   - `-2` - disables SetWriteDeadline calls completely.
	WriteTimeout string `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout" default:"3s"`

	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns" toml:"min_idle_conns"`
	// Maximum number of idle connections.
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns" toml:"max_idle_conns"`
	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Should be less than server's timeout.
	//
	// Expired connections may be closed lazily before reuse.
	// If d <= 0, connections are not closed due to a connection's idle time.
	//
	// Default is 30 minutes. -1 disables idle timeout check.
	ConnMaxIdleTime string `yaml:"conn_max_idle_time" json:"conn_max_idle_time" toml:"conn_max_idle_time" default:"30m"`
	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	//
	// Expired connections may be closed lazily before reuse.
	// If <= 0, connections are not closed due to a connection's age.
	//
	// Default is to not close idle connections.
	ConnMaxLifetime string `yaml:"conn_max_life_time" json:"conn_max_life_time" toml:"conn_max_life_time"`

	// Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int `yaml:"max_retries" json:"max_retries" toml:"max_retries" default:"3"`
	// Minimum backoff between each retry.
	// Default is 8 milliseconds; -1 disables backoff.
	MinRetryBackoff string `yaml:"min_retry_backoff" json:"min_retry_backoff" toml:"min_retry_backoff" default:"8ms"`
	// Maximum backoff between each retry.
	// Default is 512 milliseconds; -1 disables backoff.
	MaxRetryBackoff string `yaml:"max_retry_backoff" json:"max_retry_backoff" toml:"max_retry_backoff" default:"512ms"`

	// Maximum number of socket connections.
	// Default is 10 connections per every available CPU as reported by runtime.GOMAXPROCS.
	PoolSize int `yaml:"pool_size" json:"pool_size" toml:"pool_size"`
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout string `yaml:"pool_timeout" json:"pool_timeout" toml:"pool_timeout"`
}

type Redis struct {
	redis redis.UniversalClient
}

func (r *Redis) GetProxy() redis.UniversalClient {
	return r.redis
}

func (r *Redis) Close() error {
	switch rdsCli := r.redis.(type) {
	case *redis.ClusterClient:
		return rdsCli.Close()
	case *redis.Client:
		return rdsCli.Close()
	default:
		return nil
	}
}

func (r *Redis) PoolStatus() *redis.PoolStats {
	switch rdsCli := r.redis.(type) {
	case *redis.ClusterClient:
		return rdsCli.PoolStats()
	case *redis.Client:
		return rdsCli.PoolStats()
	default:
		return new(redis.PoolStats)
	}
}
