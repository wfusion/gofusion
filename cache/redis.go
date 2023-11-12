package cache

import (
	"context"
	"syscall"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/redis"

	rdsDrv "github.com/redis/go-redis/v9"
)

type rds struct {
	appName  string
	name     string
	log      log.Loggable
	instance rdsDrv.UniversalClient
}

func newRedis(appName, name string, log log.Loggable) provider {
	return &rds{
		appName:  appName,
		name:     name,
		log:      log,
		instance: redis.Use(context.Background(), name, redis.AppName(appName)),
	}
}

func (r *rds) get(ctx context.Context, keys ...string) (cached map[string]any, missed []string) {
	cached = make(map[string]any, len(keys))
	missed = make([]string, 0, len(keys))

	rs, err := r.instance.MGet(ctx, keys...).Result()
	if err != nil {
		missed = keys
		if r.log != nil {
			r.log.Info(ctx, "%v [Gofusion] %s call redis mget failed when get cache from redis "+
				"[err[%s] redis[%s]]", syscall.Getpid(), config.ComponentCache, err, r.name)
		}
		return
	}
	for i := 0; i < len(keys); i++ {
		if rs[i] != nil {
			cached[keys[i]] = rs[i]
		} else {
			missed = append(missed, keys[i])
		}
	}

	return
}

func (r *rds) set(ctx context.Context, kvs map[string]any, expired map[string]time.Duration) (failure []string) {
	failure = make([]string, 0, len(kvs))
	pipe := r.instance.Pipeline()
	keys := make([]string, 0, len(kvs))
	for k, v := range kvs {
		keys = append(keys, k)
		pipe.Set(ctx, k, v, expired[k])
	}

	cmds := utils.Must(pipe.Exec(ctx))
	for i := 0; i < len(keys); i++ {
		if err := cmds[i].Err(); err != nil {
			failure = append(failure, keys[i])
			if r.log != nil {
				r.log.Info(ctx, "%v [Gofusion] %s call redis set failed when set cache into redis "+
					"[err[%s] redis[%s] key[%s]]", syscall.Getpid(), config.ComponentCache, err, r.name, keys[i])
			}
		}
	}
	return
}

func (r *rds) del(ctx context.Context, keys ...string) (failure []string) {
	affected, err := r.instance.Del(ctx, keys...).Result()
	if err != nil && r.log != nil {
		r.log.Info(ctx, "%v [Gofusion] %s call redis del failed when delete from cache "+
			"[err[%s] redis[%s] keys%v]", syscall.Getpid(), config.ComponentCache, err, r.name, keys)
	}

	if affected == int64(len(keys)) {
		return
	}

	failure = make([]string, 0, len(keys))
	pipe := r.instance.Pipeline()
	for i := 0; i < len(keys); i++ {
		pipe.Exists(ctx, keys[i])
	}
	cmds, err := pipe.Exec(ctx)
	if err != nil && r.log != nil {
		r.log.Info(ctx, "%v [Gofusion] %s call redis exists failed when delete from cache "+
			"[err[%s] redis[%s] keys%v]", syscall.Getpid(), config.ComponentCache, err, r.name, keys)
		return keys // we cannot know whether ths keys are deleted
	}

	for i := 0; i < len(keys); i++ {
		if err := cmds[i].Err(); err != nil {
			failure = append(failure, keys[i]) // we cannot know whether ths key is deleted
			r.log.Info(ctx, "%v [Gofusion] %s call redis exists failed when delete from cache "+
				"[err[%s] redis[%s] key[%s]]", syscall.Getpid(), config.ComponentCache, err, r.name, keys[i])
			continue
		}

		if cmds[i].String() == "1" {
			failure = append(failure, keys[i])
		}
	}

	return
}
