package cache

import (
	"context"
	"errors"
	"syscall"
	"time"

	"github.com/bluele/gcache"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

func newGCache(size int, strategy string, log log.Logable) *gCache {
	cacheBuilder := gcache.New(size)
	switch strategy {
	case gcache.TYPE_ARC:
		cacheBuilder = cacheBuilder.ARC()
	case gcache.TYPE_LFU:
		cacheBuilder = cacheBuilder.LFU()
	case gcache.TYPE_LRU:
		cacheBuilder = cacheBuilder.LRU()
	case gcache.TYPE_SIMPLE:
		cacheBuilder = cacheBuilder.Simple()
	default:
		cacheBuilder = cacheBuilder.ARC()
	}

	return &gCache{
		log:      log,
		instance: cacheBuilder.Build(),
	}
}

type gCache struct {
	log      log.Logable
	instance gcache.Cache
}

func (g *gCache) get(ctx context.Context, keys ...string) (cached map[string]any, missed []string) {
	cached = make(map[string]any, len(keys))
	missed = make([]string, 0, len(keys))
	for _, k := range keys {
		v, err := g.instance.Get(k)
		if err != nil {
			missed = append(missed, k)
			if !errors.Is(err, gcache.KeyNotFoundError) && g.log != nil {
				g.log.Info(ctx, "%v [Gofusion] %s call gcache get failed "+
					"when get cache from gcache [err[%s] key[%s]]", syscall.Getpid(), config.ComponentCache, err, k)
			}
			continue
		}
		cached[k] = v
	}
	return
}

func (g *gCache) set(ctx context.Context, kvs map[string]any, expired map[string]time.Duration) (failure []string) {
	failure = make([]string, 0, len(kvs))
	for k, v := range kvs {
		var err error
		if exp, ok := expired[k]; ok {
			err = g.instance.SetWithExpire(k, v, exp)
		} else {
			err = g.instance.Set(k, v)
		}

		if err != nil {
			failure = append(failure, k)
			if g.log != nil {
				g.log.Info(ctx, "%v [Gofusion] %s call gcache set/set_with_expire failed "+
					"when set cache into gcache [err[%s] key[%s]]", syscall.Getpid(), config.ComponentCache, err, k)
			}
		}
	}
	return
}

func (g *gCache) del(ctx context.Context, keys ...string) (failure []string) {
	failure = make([]string, 0, len(keys))
	for _, k := range keys {
		if g.instance.Remove(k) {
			continue
		}

		if _, err := g.instance.Get(k); !errors.Is(err, gcache.KeyNotFoundError) {
			failure = append(failure, k)
			if g.log != nil {
				g.log.Info(ctx, "%v [Gofusion] %s call gcache remove failed "+
					"when delete cache from gcache [err[%s] key[%s]]", syscall.Getpid(), config.ComponentCache, err, k)
			}
		}
	}
	return
}
