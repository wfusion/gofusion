package cache

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constraint"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"

	pd "github.com/wfusion/gofusion/internal/util/payload"
)

type provider interface {
	get(ctx context.Context, keys ...string) (cached map[string]any, missed []string)
	set(ctx context.Context, kvs map[string]any, expired map[string]time.Duration) (failure []string)
	del(ctx context.Context, keys ...string) (failure []string)
}

type parsedConf[K constraint.Sortable, T any] struct {
	size    int
	expired time.Duration
	version int

	cacheType      cacheType
	remoteType     remoteType
	remoteInstance string
	localEvictType string
	serializeType  serialize.Algorithm
	compressType   compress.Algorithm

	log      log.Loggable
	callback callback[K, T]
}

type initOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[initOption] {
	return func(o *initOption) {
		o.appName = name
	}
}

func New[K constraint.Sortable, T any, TS ~[]T](name string, opts ...utils.OptionExtender) Cachable[K, T, TS] {
	opt := utils.ApplyOptions[initOption](opts...)

	instance := &cache[K, T, TS]{
		name:    name,
		appName: opt.appName,
		prefix:  fmt.Sprintf("%s:%s", config.Use(opt.appName).AppName(), name),
		visited: utils.NewSet[string](),
	}

	conf := instance.getConfig()
	switch conf.cacheType {
	case cacheTypeLocal:
		instance.provider = newGCache(conf.size, conf.localEvictType, conf.log)
	case cacheTypeRemote:
		if conf.remoteType != remoteTypeRedis {
			panic(UnknownRemoteType)
		}
		if !conf.serializeType.IsValid() && !conf.compressType.IsValid() {
			panic(UnknownSerializeType)
		}
		instance.provider = newRedis(opt.appName, conf.remoteInstance, conf.log)
	case cacheTypeRemoteLocal:
		panic(ErrNotImplement)
	default:
		panic(UnknownCacheType)
	}

	return instance
}

type cache[K constraint.Sortable, T any, TS ~[]T] struct {
	appName  string
	name     string
	prefix   string
	provider provider
	visited  *utils.Set[string]
}

func (c *cache[K, T, TS]) Get(ctx context.Context, keys []K, cb callback[K, T]) (ts TS) {
	conf := c.getConfig()
	innerKeys := c.convKeysToInner(keys, conf.version)
	cached, missed := c.provider.get(ctx, innerKeys...)
	kvs, _ := c.convInnerToMap(ctx, cached, conf)
	defer c.visited.Insert(innerKeys...)

	if cb == nil {
		cb = conf.callback
	}
	if len(missed) > 0 && cb != nil {
		keys := c.convInnerToKeys(missed, conf.version)
		if conf.log != nil {
			conf.log.Debug(ctx, "%v [Gofusion] %s call callback function because we do not hit the cache "+
				"when get [keys%+v]", syscall.Getpid(), config.ComponentCache, keys)
		}

		callbackKVs, opts := cb(ctx, keys)
		kvs = utils.MapMerge(kvs, callbackKVs)
		innerVals, _ := c.convMapToInner(ctx, callbackKVs, conf)
		_ = c.provider.set(ctx, innerVals, c.parseCallbackOption(kvs, conf, opts...))
		// innerFailureKeys = append(innerFailureKeys, convInnerFailureKeys...)
	}

	// order by param -> keys
	ts = make(TS, 0, len(kvs))
	missedKeys := make([]K, 0, len(keys))
	for _, k := range keys {
		v, ok := kvs[k]
		if !ok {
			missedKeys = append(missedKeys, k)
			continue
		}

		ts = append(ts, v)
	}
	if len(missedKeys) > 0 && conf.log != nil {
		conf.log.Info(ctx, "%v [Gofusion] %s we still get missing keys after callback when cache get [keys%v]",
			syscall.Getpid(), config.ComponentCache, missedKeys)
	}

	return
}

func (c *cache[K, T, TS]) GetAll(ctx context.Context, cb callback[K, T]) (ts TS) {
	conf := c.getConfig()
	allInnerKeys := c.visited.Items()
	cached, missed := c.provider.get(ctx, allInnerKeys...)
	kvs, _ := c.convInnerToMap(ctx, cached, conf)
	if len(missed) > 0 && (cb != nil || conf.callback != nil) {
		keys := c.convInnerToKeys(missed, conf.version)
		if conf.log != nil {
			conf.log.Info(ctx, "%v [Gofusion] %s call callback function because we do not hit the cache "+
				"when get all [keys%+v]", syscall.Getpid(), config.ComponentCache, keys)
		}

		var (
			callbackKVs map[K]T
			opts        []utils.OptionExtender
		)
		if cb != nil {
			callbackKVs, opts = cb(ctx, keys)
		} else {
			callbackKVs, opts = conf.callback(ctx, keys)
		}

		kvs = utils.MapMerge(kvs, callbackKVs)

		innerVals, _ := c.convMapToInner(ctx, callbackKVs, conf)
		c.provider.set(ctx, innerVals, c.parseCallbackOption(kvs, conf, opts...))
	}

	// order by param -> keys
	ts = make(TS, 0, len(kvs))
	missedKeys := make([]K, 0, len(allInnerKeys))
	for _, k := range c.convInnerToKeys(allInnerKeys, conf.version) {
		v, ok := kvs[k]
		if !ok {
			missedKeys = append(missedKeys, k)
			continue
		}

		ts = append(ts, v)
	}
	c.visited.Remove(c.convKeysToInner(missedKeys, conf.version)...)
	if len(missedKeys) > 0 && conf.log != nil {
		conf.log.Warn(ctx, "%v [Gofusion] %s index key value failed when cache get all [keys%v]",
			syscall.Getpid(), config.ComponentCache, missedKeys)
	}

	return
}

func (c *cache[K, T, TS]) Set(ctx context.Context, kvs map[K]T, opts ...utils.OptionExtender) (failure []K) {
	conf := c.getConfig()
	innerVals, innerFailureKeys := c.convMapToInner(ctx, kvs, conf)
	defer func() {
		innerKeys := utils.MapKeys(innerVals)
		c.visited.Insert(innerKeys...)
	}()

	innerFailureKeys = append(
		innerFailureKeys,
		c.provider.set(ctx, innerVals, c.parseCallbackOption(kvs, conf, opts...))...,
	)

	if failure = c.convInnerToKeys(innerFailureKeys, conf.version); len(failure) > 0 && conf.log != nil {
		conf.log.Info(ctx, "%v [Gofusion] %s set some kvs failed when set [keys%+v vals%+v]",
			syscall.Getpid(), config.ComponentCache, failure, utils.MapValuesByKeys(kvs, failure))
	}

	return
}

func (c *cache[K, T, TS]) Del(ctx context.Context, keys ...K) (failure []K) {
	conf := c.getConfig()
	innerKeys := c.convKeysToInner(keys, conf.version)
	innerFailureKeys := c.provider.del(ctx, innerKeys...)
	defer c.visited.Remove(innerKeys...)

	if failure = c.convInnerToKeys(innerFailureKeys, conf.version); len(failure) > 0 && conf.log != nil {
		conf.log.Info(ctx, "%v [Gofusion] %s del some kvs failed when del [keys%+v]",
			syscall.Getpid(), config.ComponentCache, failure)
	}
	return
}

func (c *cache[K, T, TS]) Clear(ctx context.Context) (failureKeys []K) {
	conf := c.getConfig()
	innerKeys := c.visited.Items()
	innerFailureKeys := c.provider.del(ctx, innerKeys...)
	defer c.visited.Remove(innerKeys...)

	if failureKeys = c.convInnerToKeys(innerFailureKeys, conf.version); len(failureKeys) > 0 && conf.log != nil {
		conf.log.Info(ctx, "%v [Gofusion] %s del some kvs failed when clear [keys%+v]",
			syscall.Getpid(), config.ComponentCache, failureKeys)
	}
	return
}

func (c *cache[K, T, TS]) convKeysToInner(keys []K, version int) (inner []string) {
	return utils.SliceMapping(keys, func(k K) string {
		return c.convKeyToInner(k, version)
	})
}

func (c *cache[K, T, TS]) convKeyToInner(k K, ver int) (inner string) {
	return fmt.Sprintf("%s:%v:%s", c.prefix, ver, cast.ToString(k))
}

func (c *cache[K, T, TS]) convInnerToKeys(innerKeys []string, ver int) (keys []K) {
	return utils.SliceMapping(innerKeys, func(inner string) K {
		return c.convInnerToKey(inner, ver)
	})
}

func (c *cache[K, T, TS]) convInnerToKey(inner string, ver int) (k K) {
	key := strings.TrimPrefix(inner, fmt.Sprintf("%s:%v:", c.prefix, ver))
	return utils.SortableToGeneric[string, K](key)
}

func (c *cache[K, T, TS]) convMapToInner(ctx context.Context, kvs map[K]T, conf *parsedConf[K, T]) (
	inner map[string]any, innerFailureKeys []string) {
	inner = make(map[string]any, len(kvs))
	for k, v := range kvs {
		innerKey := c.convKeyToInner(k, conf.version)
		innerVal, err := c.convValToInner(v, conf)
		if err != nil {
			if conf.log != nil {
				conf.log.Info(ctx, "%v [Gofusion] %s convert value to inner failed [err[%s] key[%+v] val[%+v]]",
					syscall.Getpid(), config.ComponentCache, err, k, v)
			}
			innerFailureKeys = append(innerFailureKeys, innerKey)
			continue
		}
		inner[innerKey] = innerVal
	}
	return
}

func (c *cache[K, T, TS]) convValToInner(src T, conf *parsedConf[K, T]) (dst any, err error) {
	if !conf.serializeType.IsValid() && !conf.compressType.IsValid() {
		return c.cloneVal(src), nil
	}

	return c.seal(src, conf)
}

func (c *cache[K, T, TS]) convInnerToMap(ctx context.Context, inner map[string]any, conf *parsedConf[K, T]) (
	kvs map[K]T, innerFailureKeys []string) {
	kvs = make(map[K]T, len(inner))
	for k, v := range inner {
		innerKey := c.convInnerToKey(k, conf.version)
		innerVal, err := c.convInnerToVal(v)
		if err != nil {
			if conf.log != nil {
				conf.log.Info(ctx, "%v [Gofusion] %s convert inner to value failed [err[%s] key[%+v] val[%+v]]",
					syscall.Getpid(), config.ComponentCache, err, innerKey, v)
			}
			innerFailureKeys = append(innerFailureKeys, k)
			continue
		}
		kvs[innerKey] = innerVal
	}
	return
}

func (c *cache[K, T, TS]) convInnerToVal(src any) (dst T, err error) {
	srcBytes, ok1 := src.([]byte)
	srcString, ok2 := src.(string)
	if !ok1 && !ok2 {
		return c.cloneVal(src.(T)), nil
	}
	if ok2 {
		buffer, cb := utils.BytesBufferPool.Get(nil)
		defer cb()
		buffer.WriteString(srcString)
		srcBytes = buffer.Bytes()
	}
	dst, ok, err := c.unseal(srcBytes)
	if err != nil {
		return
	}
	if !ok {
		return c.cloneVal(src.(T)), nil
	}
	return
}

func (c *cache[K, T, TS]) parseCallbackOption(kvs map[K]T, conf *parsedConf[K, T], opts ...utils.OptionExtender) (
	exp map[string]time.Duration) {
	opt := utils.ApplyOptions[option[K]](opts...)
	exp = make(map[string]time.Duration, len(kvs))

	// opt.keyExpired > opt.expired > conf.expired
	if conf.expired > 0 {
		for k := range kvs {
			innerKey := c.convKeyToInner(k, conf.version)
			exp[innerKey] = conf.expired
		}
	}

	if opt.expired > 0 {
		for k := range kvs {
			innerKey := c.convKeyToInner(k, conf.version)
			exp[innerKey] = conf.expired
		}
	}

	for k, e := range opt.keyExpired {
		exp[c.convKeyToInner(k, conf.version)] = e
	}

	return
}

func (c *cache[K, T, TS]) seal(src T, conf *parsedConf[K, T]) (dst []byte, err error) {
	return pd.Seal(src, pd.Serialize(conf.serializeType), pd.Compress(conf.compressType))
}

func (c *cache[K, T, TS]) unseal(src []byte) (dst T, ok bool, err error) {
	_, dst, ok, err = pd.UnsealT[T](src)
	return
}

func (c *cache[K, T, TS]) cloneVal(src T) (dst T) {
	if cl, ok := any(src).(clone.Clonable[T]); ok {
		dst = cl.Clone()
		return
	}
	return clone.Slowly(src)
}

func (c *cache[K, T, TS]) getConfig() (conf *parsedConf[K, T]) {
	var cfgs map[string]*Conf
	_ = config.Use(c.appName).LoadComponentConfig(config.ComponentCache, &cfgs)
	if len(cfgs) == 0 {
		panic(ErrCacheNotFound)
	}

	cfg, ok := cfgs[c.name]
	if !ok {
		panic(ErrCacheNotFound)
	}

	conf = &parsedConf[K, T]{
		size:           cfg.Size,
		localEvictType: cfg.LocalEvictType,
		cacheType:      cfg.CacheType,
		remoteType:     cfg.RemoteType,
		remoteInstance: cfg.RemoteInstance,
		version:        cfg.Version,
	}
	if utils.IsStrNotBlank(cfg.Expired) {
		conf.expired = utils.Must(time.ParseDuration(cfg.Expired))
	}
	if utils.IsStrNotBlank(cfg.LogInstance) {
		conf.log = log.Use(cfg.LogInstance, log.AppName(c.appName))
	}

	if utils.IsStrNotBlank(cfg.SerializeType) {
		conf.serializeType = serialize.ParseAlgorithm(cfg.SerializeType)
	}

	if utils.IsStrNotBlank(cfg.Compress) {
		conf.compressType = compress.ParseAlgorithm(cfg.Compress)
		if !conf.compressType.IsValid() {
			panic(UnknownCompress)
		}
	}
	// default serialize type when need compress
	if conf.compressType.IsValid() && !conf.serializeType.IsValid() {
		conf.serializeType = serialize.AlgorithmGob
	}

	if utils.IsStrNotBlank(cfg.Callback) {
		callbackFn := inspect.FuncOf(cfg.Callback)
		if callbackFn == nil {
			panic(errors.Errorf("not found callback function: %s", cfg.Callback))
		}
		conf.callback = *(*func(ctx context.Context, missed []K) (rs map[K]T, opts []utils.OptionExtender))(callbackFn)
	}

	return
}
