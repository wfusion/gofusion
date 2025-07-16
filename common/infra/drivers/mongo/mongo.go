package mongo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
)

// Option
//nolint: revive // mongo options comments too long issue
type Option struct {
	DB        string   `yaml:"db" json:"db" toml:"db"`
	AuthDB    string   `yaml:"auth_db" json:"auth_db" toml:"auth_db" default:"admin"`
	User      string   `yaml:"user" json:"user" toml:"user"`
	Password  string   `yaml:"password" json:"password" toml:"password" encrypted:""`
	Endpoints []string `yaml:"endpoints" json:"endpoints" toml:"endpoints"`

	// Timeout specifies the amount of time that a single operation run on this Client can execute before returning an error.
	// The deadline of any operation run through the Client will be honored above any Timeout set on the Client; Timeout will only
	// be honored if there is no deadline on the operation Context. Timeout can also be set through the "timeoutMS" URI option
	// (e.g. "timeoutMS=1000"). The default value is nil, meaning operations do not inherit a timeout from the Client.
	//
	// If any Timeout is set (even 0) on the Client, the values of MaxTime on operation options, TransactionOptions.MaxCommitTime and
	// SessionOptions.DefaultMaxCommitTime will be ignored. Setting Timeout and SocketTimeout or writeConcern.wTimeout will result
	// in undefined behavior.
	//
	// NOTE(benjirewis): SetTimeout represents unstable, provisional API. The behavior of the driver when a Timeout is specified is
	// subject to change.
	Timeout string `yaml:"timeout" json:"timeout" toml:"timeout" default:"5s"`
	// ConnTimeout specifies a timeout that is used for creating connections to the server. If a custom Dialer is
	// specified through SetDialer, this option must not be used. This can be set through ApplyURI with the
	// "connectTimeoutMS" (e.g "connectTimeoutMS=30") option. If set to 0, no timeout will be used. The default is 30
	// seconds.
	ConnTimeout string `yaml:"conn_timeout" json:"conn_timeout" toml:"conn_timeout" default:"30s"`
	// SocketTimeout specifies the timeout to be used for the Client's socket reads and writes.
	//
	// NOTE(benjirewis): SocketTimeout will be deprecated in a future release. The more general Timeout option
	// may be used in its place to control the amount of time that a single operation can run before returning
	// an error. Setting SocketTimeout and Timeout on a single client will result in undefined behavior.
	SocketTimeout string `yaml:"socket_timeout" json:"socket_timeout" toml:"socket_timeout" default:"5s"`
	// HeartbeatInterval specifies the amount of time to wait between periodic background server checks. This can also be
	// set through the "heartbeatIntervalMS" URI option (e.g. "heartbeatIntervalMS=10000"). The default is 10 seconds.
	HeartbeatInterval string `yaml:"heartbeat_interval" json:"heartbeat_interval" toml:"heartbeat_interval" default:"10s"`

	// MaxConnecting specifies the maximum number of connections a connection pool may establish simultaneously. This can
	// also be set through the "maxConnecting" URI option (e.g. "maxConnecting=2"). If this is 0, the default is used. The
	// default is 2. Values greater than 100 are not recommended.
	MaxConnecting uint64 `yaml:"max_connecting" json:"max_connecting" toml:"max_connecting" default:"2"`
	// MinPoolSize specifies the minimum number of connections allowed in the driver's connection pool to each server. If
	// this is non-zero, each server's pool will be maintained in the background to ensure that the size does not fall below
	// the minimum. This can also be set through the "minPoolSize" URI option (e.g. "minPoolSize=100"). The default is 0.
	MinPoolSize uint64 `yaml:"min_pool_size" json:"min_pool_size" toml:"min_pool_size"`
	// MaxPoolSize specifies that maximum number of connections allowed in the driver's connection pool to each server.
	// Requests to a server will block if this maximum is reached. This can also be set through the "maxPoolSize" URI option
	// (e.g. "maxPoolSize=100"). If this is 0, maximum connection pool size is not limited. The default is 100.
	MaxPoolSize uint64 `yaml:"max_pool_size" json:"max_pool_size" toml:"max_pool_size" default:"100"`
	// MaxConnIdleTime specifies the maximum amount of time that a connection will remain idle in a connection pool
	// before it is removed from the pool and closed. This can also be set through the "maxIdleTimeMS" URI option (e.g.
	// "maxIdleTimeMS=10000"). The default is 0, meaning a connection can remain unused indefinitely.
	MaxConnIdleTime string `yaml:"max_conn_idle_time" json:"max_conn_idle_time" toml:"max_conn_idle_time" default:"10s"`

	// RetryWrites specifies whether supported write operations should be retried once on certain errors, such as network
	// errors.
	//
	// Supported operations are InsertOne, UpdateOne, ReplaceOne, DeleteOne, FindOneAndDelete, FindOneAndReplace,
	// FindOneAndDelete, InsertMany, and BulkWrite. Note that BulkWrite requests must not include UpdateManyModel or
	// DeleteManyModel instances to be considered retryable. Unacknowledged writes will not be retried, even if this option
	// is set to true.
	//
	// This option requires server version >= 3.6 and a replica set or sharded cluster and will be ignored for any other
	// cluster type. This can also be set through the "retryWrites" URI option (e.g. "retryWrites=true"). The default is
	// true.
	RetryWrites bool `yaml:"retry_writes" json:"retry_writes" toml:"retry_writes" default:"true"`

	// SetRetryReads specifies whether supported read operations should be retried once on certain errors, such as network
	// errors.
	//
	// Supported operations are Find, FindOne, Aggregate without a $out stage, Distinct, CountDocuments,
	// EstimatedDocumentCount, Watch (for Client, Database, and Collection), ListCollections, and ListDatabases. Note that
	// operations run through RunCommand are not retried.
	//
	// This option requires server version >= 3.6 and driver version >= 1.1.0. The default is true.
	RetryReads bool `yaml:"retry_reads" json:"retry_reads" toml:"retry_reads" default:"true"`
}

var Default Dialect = new(defaultDialect)

type defaultDialect struct{}

func (d *defaultDialect) New(ctx context.Context, option Option, opts ...utils.OptionExtender) (cli *Mongo, err error) {
	opt := options.Client().ApplyURI(d.parseOption(option))
	opt.SetRetryReads(option.RetryReads)
	opt.SetRetryWrites(option.RetryWrites)
	d.wrapDurationSetter(option.Timeout, func(du time.Duration) { opt.SetTimeout(du) })
	d.wrapDurationSetter(option.ConnTimeout, func(du time.Duration) { opt.SetConnectTimeout(du) })
	d.wrapDurationSetter(option.SocketTimeout, func(du time.Duration) { opt.SetSocketTimeout(du) })
	d.wrapDurationSetter(option.MaxConnIdleTime, func(du time.Duration) { opt.SetMaxConnIdleTime(du) })
	d.wrapDurationSetter(option.HeartbeatInterval, func(du time.Duration) { opt.SetHeartbeatInterval(du) })
	d.wrapNumberSetter(option.MaxConnecting, func(nu uint64) { opt.SetMaxConnecting(option.MaxConnecting) })
	d.wrapNumberSetter(option.MinPoolSize, func(nu uint64) { opt.SetMinPoolSize(option.MinPoolSize) })
	d.wrapNumberSetter(option.MaxPoolSize, func(nu uint64) { opt.SetMaxPoolSize(option.MaxPoolSize) })

	newOpt := utils.ApplyOptions[newOption](opts...)
	if newOpt.monitor != nil {
		opt = opt.SetMonitor(newOpt.monitor)
	}
	if newOpt.poolMonitor != nil {
		opt = opt.SetPoolMonitor(newOpt.poolMonitor)
	}

	mgoCli, err := mongo.Connect(ctx, opt)
	if err != nil {
		return
	}

	// authentication check
	if err = mgoCli.Ping(ctx, nil); err != nil {
		return
	}

	cli = &Mongo{Client: mgoCli}
	return
}

func (d *defaultDialect) wrapDurationSetter(s string, setter func(du time.Duration)) {
	if utils.IsStrBlank(s) {
		return
	}
	duration, err := utils.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	setter(duration)
}

func (d *defaultDialect) wrapNumberSetter(n uint64, setter func(nu uint64)) {
	if n > 0 {
		setter(n)
	}
}

func (d *defaultDialect) parseOption(option Option) (dsn string) {
	return fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=%s",
		option.User, option.Password, strings.Join(option.Endpoints, constant.Comma), option.DB, option.AuthDB)
}
