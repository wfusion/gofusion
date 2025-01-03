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
				hookValue.Interface().(redisCustomLogger).Init(logger, opt.AppName, name)
			}

			hooks = append(hooks, hookValue.Interface().(rdsDrv.Hook))
		}
	}

	ropt := redis.Option{
		Cluster:         conf.Endpoint.Cluster,
		Endpoints:       conf.Endpoint.Addresses,
		DB:              conf.Endpoint.DB,
		User:            conf.Endpoint.User,
		Password:        conf.Endpoint.Password,
		DialTimeout:     conf.Endpoint.DialTimeout,
		ReadTimeout:     conf.Endpoint.ReadTimeout,
		WriteTimeout:    conf.Endpoint.WriteTimeout,
		MinIdleConns:    conf.Endpoint.MinIdleConns,
		MaxIdleConns:    conf.Endpoint.MaxIdleConns,
		ConnMaxIdleTime: conf.Endpoint.ConnMaxIdleTime,
		ConnMaxLifetime: conf.Endpoint.ConnMaxLifetime,
		MaxRetries:      conf.Endpoint.MaxRetries,
		MinRetryBackoff: conf.Endpoint.MinRetryBackoff,
		MaxRetryBackoff: conf.Endpoint.MaxRetryBackoff,
		PoolSize:        conf.Endpoint.PoolSize,
		PoolTimeout:     conf.Endpoint.PoolTimeout,
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
		},
	}
}

func (r *redisKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Value {
	stringResult := r.cli.GetProxy().Get(ctx, key)
	return &redisValue{typ: stringRedisValueType, string: stringResult}
}

func (r *redisKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Value {
	opt := utils.ApplyOptions[setOption](opts...)
	statusResult := r.cli.GetProxy().Set(ctx, key, val, opt.expired)
	return &redisValue{typ: statusRedisValueType, status: statusResult}
}

func (r *redisKV) getProxy() any { return r.cli }
func (r *redisKV) close() error  { return r.cli.Close() }

type redisValue struct {
	typ    redisValueType
	string *rdsDrv.StringCmd
	status *rdsDrv.StatusCmd
}

func (r *redisValue) String() (string, error) {
	if r == nil {
		return "", ErrNilValue
	}
	switch r.typ {
	case stringRedisValueType:
		return r.string.Result()
	case statusRedisValueType:
		return r.status.Result()
	default:
		return "", ErrUnsupportedRedisValueType
	}
}

type redisValueType int

const (
	stringRedisValueType redisValueType = 1 + iota
	intRedisValueType
	floatRedisValueType
	boolRedisValueType
	stringSliceRedisValueType
	statusRedisValueType
	durationRedisValueType
)
