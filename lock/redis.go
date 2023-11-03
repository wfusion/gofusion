package lock

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/redis"

	rdsDrv "github.com/redis/go-redis/v9"
)

const (
	redisLuaLockCommand = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
    return "OK"
else
    return redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
end`

	redisLuaUnlockCommand = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`
)

type redisLuaLocker struct {
	ctx       context.Context
	appName   string
	redisName string
}

func newRedisLuaLocker(ctx context.Context, appName, redisName string) ReentrantLockable {
	return &redisLuaLocker{ctx: ctx, appName: appName, redisName: redisName}
}

func (r *redisLuaLocker) Lock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	if utils.IsStrBlank(opt.reentrantKey) {
		return ErrReentrantKeyNotFound
	}
	expired := tolerance
	if opt.expired > 0 {
		expired = opt.expired
	}
	lockKey := r.formatLockKey(key)
	err = redis.
		Use(ctx, r.redisName, redis.AppName(r.appName)).
		Eval(ctx, redisLuaLockCommand, []string{lockKey}, []string{
			opt.reentrantKey, strconv.Itoa(int(expired / time.Millisecond)),
		}).
		Err()
	if errors.Is(err, rdsDrv.Nil) {
		err = ErrTimeout
	}
	return
}

func (r *redisLuaLocker) Unlock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	if utils.IsStrBlank(opt.reentrantKey) {
		return ErrReentrantKeyNotFound
	}
	lockKey := r.formatLockKey(key)
	return redis.
		Use(ctx, r.redisName, redis.AppName(r.appName)).
		Eval(ctx, redisLuaUnlockCommand, []string{lockKey}, []string{
			opt.reentrantKey, strconv.Itoa(int(opt.expired / time.Millisecond)),
		}).
		Err()
}

func (r *redisLuaLocker) ReentrantLock(ctx context.Context, key, reentrantKey string,
	opts ...utils.OptionExtender) (err error) {
	return r.Lock(ctx, key, append(opts, ReentrantKey(reentrantKey))...)
}

func (r *redisLuaLocker) formatLockKey(key string) (format string) {
	return fmt.Sprintf("%s:%s", config.Use(r.appName).AppName(), key)
}

type redisNXLocker struct {
	ctx       context.Context
	appName   string
	redisName string
}

func newRedisNXLocker(ctx context.Context, appName, redisName string) Lockable {
	return &redisNXLocker{ctx: ctx, appName: appName, redisName: redisName}
}

func (r *redisNXLocker) Lock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	expired := tolerance
	if opt.expired > 0 {
		expired = opt.expired
	}
	lockKey := r.formatLockKey(key)
	cmd := redis.Use(ctx, r.redisName, redis.AppName(r.appName)).SetNX(ctx, lockKey, utils.UUID(), expired)
	if err = cmd.Err(); err != nil {
		return
	}
	if !cmd.Val() {
		err = ErrTimeout
		return
	}
	return
}

func (r *redisNXLocker) Unlock(ctx context.Context, key string, _ ...utils.OptionExtender) (err error) {
	lockKey := r.formatLockKey(key)
	return redis.Use(ctx, r.redisName, redis.AppName(r.appName)).Del(ctx, lockKey).Err()
}

func (r *redisNXLocker) formatLockKey(key string) (format string) {
	return fmt.Sprintf("%s:%s", config.Use(r.appName).AppName(), key)
}
