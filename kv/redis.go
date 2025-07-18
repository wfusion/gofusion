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

func newRedisInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storable {
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
		DialTimeout:     conf.Endpoint.DialTimeout.String(),
		ReadTimeout:     conf.Endpoint.RedisReadTimeout.String(),
		WriteTimeout:    conf.Endpoint.RedisWriteTimeout.String(),
		MinIdleConns:    conf.Endpoint.RedisMinIdleConns,
		MaxIdleConns:    conf.Endpoint.RedisMaxIdleConns,
		ConnMaxIdleTime: conf.Endpoint.RedisConnMaxIdleTime.String(),
		ConnMaxLifetime: conf.Endpoint.RedisConnMaxLifetime.String(),
		MaxRetries:      conf.Endpoint.RedisMaxRetries,
		MinRetryBackoff: conf.Endpoint.RedisMinRetryBackoff.String(),
		MaxRetryBackoff: conf.Endpoint.RedisMaxRetryBackoff.String(),
		PoolSize:        conf.Endpoint.RedisPoolSize,
		PoolTimeout:     conf.Endpoint.RedisPoolTimeout.String(),
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

func (r *redisKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Got {
	opt := utils.ApplyOptions[option](opts...)
	if !opt.withPrefix {
		if !opt.withKeysOnly {
			return &redisGetValue{StringCmd: r.cli.GetProxy().Get(ctx, key)}
		} else {
			intCmd := r.cli.GetProxy().Exists(ctx, key)
			cmd := rdsDrv.NewStringCmd(ctx, intCmd.Args()...)
			cmd.SetErr(intCmd.Err())
			return &redisGetValue{StringCmd: cmd}
		}
	}

	pattern := key
	if !strings.Contains(key, "*") {
		pattern += "*"
	}

	result := new(redisGetValue)
	keysResult := r.cli.GetProxy().Keys(ctx, pattern)
	result.StringCmd = rdsDrv.NewStringCmd(ctx, keysResult.Args()...)
	if keysResult.Err() != nil {
		result.StringCmd.SetErr(keysResult.Err())
		return result
	}
	keys := keysResult.Val()
	if len(keys) == 0 {
		result.StringCmd.SetErr(rdsDrv.Nil)
		return result
	}

	if opt.withKeysOnly {
		result.multi = make(map[string]any)
		for _, key := range keys {
			result.multi[key] = nil
		}
		return result
	}

	mgetResult := r.cli.GetProxy().MGet(ctx, keys...)
	result.StringCmd = rdsDrv.NewStringCmd(ctx, mgetResult.Args()...)
	if mgetResult.Err() != nil {
		result.StringCmd.SetErr(mgetResult.Err())
		return result
	}
	vals := mgetResult.Val()
	result.multi = make(map[string]any, len(keys))
	length := utils.Min(len(keys), len(vals))
	for i := 0; i < length; i++ {
		result.multi[keys[i]] = vals[i]
	}
	return result
}

func (r *redisKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Put {
	opt := utils.ApplyOptions[option](opts...)
	return &redisPutValue{StatusCmd: r.cli.GetProxy().Set(ctx, key, val, opt.expired), key: key}
}

func (r *redisKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) Del {
	opt := utils.ApplyOptions[option](opts...)
	if !opt.withPrefix {
		return &redisDelValue{IntCmd: r.cli.GetProxy().Del(ctx, key)}
	}
	pattern := key
	if !strings.Contains(key, "*") {
		pattern += "*"
	}

	keysResult := r.cli.GetProxy().Keys(ctx, pattern)
	if keysResult.Err() != nil {
		cmd := rdsDrv.NewIntCmd(ctx, keysResult.Args()...)
		cmd.SetErr(keysResult.Err())
		return &redisDelValue{IntCmd: cmd}
	}

	keys := keysResult.Val()
	if len(keys) == 0 {
		return &redisDelValue{IntCmd: rdsDrv.NewIntCmd(ctx, keysResult.Args()...)}
	}
	return &redisDelValue{IntCmd: r.cli.GetProxy().Del(ctx, keys...)}
}

func (r *redisKV) Has(ctx context.Context, key string, opts ...utils.OptionExtender) Had {
	opt := utils.ApplyOptions[option](opts...)
	if !opt.withPrefix {
		return &redisExistsValue{IntCmd: r.cli.GetProxy().Exists(ctx, key), key: key}
	}
	pattern := key
	if !strings.Contains(key, "*") {
		pattern += "*"
	}
	cmd := rdsDrv.NewIntCmd(ctx, pattern)
	iter := r.Paginate(ctx, pattern, 100, KeysOnly())
	for iter.More() {
		kvs, err := iter.Next()
		if err != nil {
			cmd.SetErr(err)
			cmd.SetVal(int64(len(kvs)))
			return &redisExistsValue{IntCmd: cmd, key: pattern}
		}
		if len(kvs) > 0 {
			cmd.SetVal(int64(len(kvs)))
			return &redisExistsValue{IntCmd: cmd, key: pattern}
		}
	}
	return &redisExistsValue{IntCmd: rdsDrv.NewIntCmd(ctx, pattern), key: pattern}
}

func (r *redisKV) Paginate(ctx context.Context, pattern string, pageSize int, opts ...utils.OptionExtender) Paginated {
	if !strings.Contains(pattern, "*") {
		pattern += "*"
	}
	opt := utils.ApplyOptions[option](opts...)

	cursor := uint64(0)
	if opt.cursor != nil {
		cursor = cast.ToUint64(opt.cursor)
	}

	return &redisPagination{
		abstractPagination: newAbstractPagination(ctx, pageSize, opt),
		kv:                 r,
		first:              true,
		pattern:            pattern,
		cursor:             cursor,
	}
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

	keys []string
}

func (r *redisDelValue) Err() error {
	if r == nil || r.IntCmd == nil {
		return ErrNilValue
	}
	return r.IntCmd.Err()
}

type redisPagination struct {
	*abstractPagination
	kv *redisKV

	first   bool
	pattern string
	cursor  uint64
}

func (r *redisPagination) More() bool {
	if r == nil {
		return false
	}
	return r.first || r.cursor > 0
}

func (r *redisPagination) Next() (kvs KeyValues, err error) {
	if r == nil {
		return nil, ErrNilValue
	}
	r.first = false
	keys, next, err := r.kv.cli.GetProxy().Scan(r.ctx, r.cursor, r.pattern, int64(r.count)).Result()
	if err != nil {
		return
	}
	r.cursor = next
	if r.opt.withKeysOnly {
		kvs = make(KeyValues, 0, len(keys))
		for _, key := range keys {
			kvs = append(kvs, &KeyValue{Key: key, Val: nil, Ver: newDefaultVersion()})
		}
		return
	}
	if len(keys) == 0 {
		return
	}
	vals, err := r.kv.cli.GetProxy().MGet(r.ctx, keys...).Result()
	if err != nil {
		return
	}
	kvs = make(KeyValues, 0, len(vals))
	length := utils.Min(len(keys), len(vals))
	for i := 0; i < length; i++ {
		kvs = append(kvs, &KeyValue{Key: keys[i], Val: vals[i], Ver: newDefaultVersion()})
	}
	return
}

func (r *redisPagination) Cursor() any {
	if r == nil {
		return nil
	}
	return r.cursor
}
