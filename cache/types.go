package cache

import (
	"context"
	"errors"
	"time"

	"github.com/wfusion/gofusion/common/constraint"
	"github.com/wfusion/gofusion/common/utils"
)

var (
	UnknownCacheType     = errors.New("unknown cache type")
	UnknownCompress      = errors.New("unknown compress type")
	UnknownSerializeType = errors.New("unknown serialize type")
	UnknownRemoteType    = errors.New("unknown remote type")
	ErrNotImplement      = errors.New("not implement")
	ErrCacheNotFound     = errors.New("not found cache to use")
	ErrCallbackNotFound  = errors.New("not found callback function")
)

type Cachable[K constraint.Sortable, T any, TS ~[]T] interface {
	Get(ctx context.Context, keys []K, cb callback[K, T]) TS
	GetAll(ctx context.Context, cb callback[K, T]) TS
	Set(ctx context.Context, kv map[K]T, opts ...utils.OptionExtender) (failure []K)
	Del(ctx context.Context, keys ...K) (failure []K)
	Clear(ctx context.Context) (failure []K)
}

type callback[K constraint.Sortable, T any] func(ctx context.Context, missed []K) (
	rs map[K]T, opts []utils.OptionExtender)

type option[K constraint.Sortable] struct {
	expired    time.Duration
	keyExpired map[K]time.Duration
}

func Expired[K constraint.Sortable](expired time.Duration) utils.OptionFunc[option[K]] {
	return func(o *option[K]) {
		o.expired = expired
	}
}

func KeyExpired[K constraint.Sortable](keyExpired map[K]time.Duration) utils.OptionFunc[option[K]] {
	return func(o *option[K]) {
		o.keyExpired = keyExpired
	}
}

type cacheType string

const (
	// cacheTypeLocal local cache, base on gcache
	cacheTypeLocal cacheType = "local"
	// cacheTypeRemote remote cache should be serialized
	cacheTypeRemote cacheType = "remote"
	// cacheTypeRemoteLocal remove cache version and local cache data, not implement now
	cacheTypeRemoteLocal cacheType = "remote_local"
)

type remoteType string

const (
	remoteTypeRedis remoteType = "redis"
)

type Conf struct {
	Size           int        `yaml:"size" json:"size" toml:"size" default:"10000"`
	Expired        string     `yaml:"expired" json:"expired" toml:"expired" default:"1h"`
	Version        int        `yaml:"version" json:"version" toml:"version"`
	CacheType      cacheType  `yaml:"type" json:"type" toml:"type" default:"local"`
	RemoteType     remoteType `yaml:"remote_type" json:"remote_type" toml:"remote_type" default:"redis"`
	RemoteInstance string     `yaml:"remote_instance" json:"remote_instance" toml:"remote_instance"`
	LocalEvictType string     `yaml:"local_evict_type" json:"local_evict_type" toml:"local_evict_type" default:"arc"`
	Compress       string     `yaml:"compress" json:"compress" toml:"compress"`
	SerializeType  string     `yaml:"serialize_type" json:"serialize_type" toml:"serialize_type"`
	Callback       string     `yaml:"callback" json:"callback" toml:"callback"`
	LogInstance    string     `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}
