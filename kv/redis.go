package kv

import (
	"context"
	"reflect"

	"github.com/wfusion/gofusion/common/infra/drivers/redis"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"

	rdsDrv "github.com/redis/go-redis/v9"

	fusLog "github.com/wfusion/gofusion/log"
)

type redisKV struct {
	abstractKV

	cli *redis.Redis
}

func newRedisInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) KeyValue {
	var hooks []rdsDrv.Hook
	for _, hookLoc := range conf.Endpoint.RedisHooks {
		if hookType := inspect.TypeOf(hookLoc); hookType != nil {
			hookValue := reflect.New(hookType)
			if hookValue.Type().Implements(redisCustomLoggerType) {
				logger := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
				hookValue.Interface().(redisCustomLogger).Init(logger, opt.AppName, name, conf.LogInstance)
			}

			hooks = append(hooks, hookValue.Interface().(rdsDrv.Hook))
		}
	}

	ropt := redis.Option{
		Cluster:         conf.Endpoint.RedisCluster,
		Endpoints:       conf.Endpoint.Addresses,
		DB:              conf.Endpoint.RedisDB,
		User:            conf.Endpoint.User,
		Password:        conf.Endpoint.Password,
		DialTimeout:     conf.Endpoint.DialTimeout,
		ReadTimeout:     conf.Endpoint.RedisReadTimeout,
		WriteTimeout:    conf.Endpoint.RedisWriteTimeout,
		MinIdleConns:    conf.Endpoint.RedisMinIdleConns,
		MaxIdleConns:    conf.Endpoint.RedisMaxIdleConns,
		ConnMaxIdleTime: conf.Endpoint.RedisConnMaxIdleTime,
		ConnMaxLifetime: conf.Endpoint.RedisConnMaxLifetime,
		MaxRetries:      conf.Endpoint.RedisMaxRetries,
		MinRetryBackoff: conf.Endpoint.RedisMinRetryBackoff,
		MaxRetryBackoff: conf.Endpoint.RedisMaxRetryBackoff,
		PoolSize:        conf.Endpoint.RedisPoolSize,
		PoolTimeout:     conf.Endpoint.RedisPoolTimeout,
	}
	cli, err := redis.Default.New(ctx, ropt, redis.WithHook(hooks))
	if err != nil {
		panic(err)
	}

	return &redisKV{
		cli: cli,
		abstractKV: abstractKV{
			ctx:     ctx,
			appName: opt.AppName,
			name:    name,
			conf:    conf,
		},
	}
}

func (r *redisKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	//opt := utils.ApplyOptions[getOption](opts...)
	return &redisGetValue{StringCmd: r.cli.GetProxy().Get(ctx, key)}
}

func (r *redisKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[setOption](opts...)
	return &redisPutValue{StatusCmd: r.cli.GetProxy().Set(ctx, key, val, opt.expired), key: key}
}

func (r *redisKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	//opt := utils.ApplyOptions[delOption](opts...)
	return &redisDelValue{IntCmd: r.cli.GetProxy().Del(ctx, key)}
}

func (r *redisKV) getProxy() any { return r.cli }
func (r *redisKV) close() error  { return r.cli.Close() }

type redisGetValue struct {
	*rdsDrv.StringCmd
}

func (r *redisGetValue) String() (string, error) {
	if r == nil {
		return "", ErrNilValue
	}
	return r.StringCmd.Result()
}

func (r *redisGetValue) Version() (Version, error) {
	if r == nil {
		return nil, ErrNilValue
	}
	return new(emptyVersion), nil
}

type redisPutValue struct {
	*rdsDrv.StatusCmd

	key string
}

func (r *redisPutValue) LeaseID() string {
	if r == nil {
		return ""
	}
	return r.key
}

func (r *redisPutValue) Err() error {
	if r == nil {
		return ErrNilValue
	}
	return r.StatusCmd.Err()
}

type redisDelValue struct {
	*rdsDrv.IntCmd
}

func (r *redisDelValue) Err() error {
	if r == nil {
		return ErrNilValue
	}
	return r.IntCmd.Err()
}
