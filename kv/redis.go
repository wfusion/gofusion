package kv

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

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

func newRedisInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storage {
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
	opt := utils.ApplyOptions[queryOption](opts...)
	if !opt.withPrefix {
		return &redisGetValue{StringCmd: r.cli.GetProxy().Get(ctx, key)}
	}

	pattern := key
	if !strings.Contains(key, "*") {
		pattern += "*"
	}
	limit := 100
	batch := int64(100)
	if opt.limit > 0 {
		limit = opt.limit
		if batch > int64(limit) {
			batch = int64(limit)
		}
	}

	var (
		multi  = make(map[string]any, limit)
		cursor = uint64(0)
	)

	for {
		keys, cursor, err := r.cli.GetProxy().Scan(ctx, cursor, pattern, batch).Result()
		if err != nil {
			cmd := rdsDrv.NewStringCmd(ctx, pattern)
			cmd.SetErr(err)
			return &redisGetValue{StringCmd: cmd}
		}
		if opt.withKeysOnly {
			for i := 0; i < len(keys); i++ {
				multi[keys[i]] = nil
			}
		} else {
			vals, err := r.cli.GetProxy().MGet(ctx, keys...).Result()
			if err != nil {
				cmd := rdsDrv.NewStringCmd(ctx, pattern)
				cmd.SetErr(err)
				return &redisGetValue{StringCmd: cmd, multi: multi}
			}
			length := utils.Min(len(keys), len(vals))
			for i := 0; i < length; i++ {
				multi[keys[i]] = vals[i]
			}
		}
		if len(keys) >= limit {
			break
		}
		if cursor == 0 {
			break
		}
	}
	return &redisGetValue{multi: multi}
}

func (r *redisKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[writeOption](opts...)
	return &redisPutValue{StatusCmd: r.cli.GetProxy().Set(ctx, key, val, opt.expired), key: key}
}

func (r *redisKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	//opt := utils.ApplyOptions[writeOption](opts...)
	return &redisDelValue{IntCmd: r.cli.GetProxy().Del(ctx, key)}
}

func (r *redisKV) Exists(ctx context.Context, key string, opts ...utils.OptionExtender) ExistsVal {
	opt := utils.ApplyOptions[queryOption](opts...)
	if !opt.withPrefix {
		return &redisExistsValue{IntCmd: r.cli.GetProxy().Exists(ctx, key), key: key}
	}
	pattern := key
	if !strings.Contains(key, "*") {
		pattern += "*"
	}

	keys, _, err := r.cli.GetProxy().Scan(ctx, 0, pattern, 1).Result()
	cmd := rdsDrv.NewIntCmd(ctx, pattern)
	cmd.SetErr(err)
	cmd.SetVal(int64(len(keys)))
	return &redisExistsValue{IntCmd: cmd, key: pattern}
}

func (r *redisKV) getProxy() any { return r.cli }
func (r *redisKV) close() error  { return r.cli.Close() }

type redisGetValue struct {
	*rdsDrv.StringCmd
	multi map[string]any
}

func (r *redisGetValue) Err() error {
	if r == nil || (r.StringCmd == nil && r.multi == nil) {
		return ErrNilValue
	}
	if r.StringCmd != nil {
		if errors.Is(rdsDrv.Nil, r.StringCmd.Err()) {
			return ErrNilValue
		}
		return r.StringCmd.Err()
	}
	return nil
}

func (r *redisGetValue) String() string {
	if r == nil {
		return ""
	}
	if r.multi != nil {
		if vals := utils.MapValues(r.multi); len(vals) > 0 {
			return cast.ToString(vals[0])
		}
		return ""
	}
	return r.StringCmd.Val()
}

func (r *redisGetValue) KeyValues() KeyValues {
	if r == nil {
		return nil
	}
	kvs := make(KeyValues, 0, len(r.multi))
	for k, v := range r.multi {
		kvs = append(kvs, &KeyValue{Key: k, Val: v, Ver: newDefaultVersion()})
	}
	return kvs
}

func (r *redisGetValue) Version() Version {
	if r == nil {
		return newEmptyVersion()
	}
	return newDefaultVersion()
}

type redisExistsValue struct {
	*rdsDrv.IntCmd

	key string
}

func (r *redisExistsValue) Bool() bool {
	if r == nil || r.IntCmd == nil {
		return false
	}
	return r.IntCmd.Val() > 0
}

func (r *redisExistsValue) Err() error {
	if r == nil || r.IntCmd == nil || errors.Is(rdsDrv.Nil, r.IntCmd.Err()) {
		return ErrNilValue
	}
	return r.IntCmd.Err()
}

func (r *redisExistsValue) Version() Version {
	if r == nil {
		return newEmptyVersion()
	}
	return newDefaultVersion()
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
