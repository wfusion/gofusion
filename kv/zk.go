package kv

import (
	"context"
	"math/big"
	"reflect"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-zookeeper/zk"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

type zkKV struct {
	abstractKV

	cli     *zk.Conn
	closeCh <-chan zk.Event
}

func newZKInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storage {
	var (
		conn *zk.Conn
		ech  <-chan zk.Event
		err  error
	)

	maxBufferSize, _ := humanize.ParseBytes(conf.Endpoint.ZooMaxBufferSize)
	connMaxBufferSize, _ := humanize.ParseBytes(conf.Endpoint.ZooMaxConnBufferSize)

	if conf.EnableLogger && utils.IsStrNotBlank(conf.Endpoint.ZooLogger) {
		logInfo := log.Use(conf.LogInstance, log.AppName(opt.AppName)).Level(ctx) < log.WarnLevel
		conn, ech, err = zk.Connect(
			conf.Endpoint.Addresses,
			utils.Must(time.ParseDuration(conf.Endpoint.DialTimeout)),
			zk.WithLogger(reflect.New(inspect.TypeOf(conf.Endpoint.ZooLogger)).Interface().(zk.Logger)),
			zk.WithLogInfo(logInfo),
			zk.WithMaxBufferSize(int(maxBufferSize)),
			zk.WithMaxConnBufferSize(int(connMaxBufferSize)),
		)
	} else {
		conn, ech, err = zk.Connect(
			conf.Endpoint.Addresses,
			utils.Must(time.ParseDuration(conf.Endpoint.DialTimeout)),
			zk.WithMaxBufferSize(int(maxBufferSize)),
			zk.WithMaxConnBufferSize(int(connMaxBufferSize)),
		)
	}

	if err != nil {
		panic(err)
	}

	return &zkKV{
		cli:     conn,
		closeCh: ech,
		abstractKV: abstractKV{
			name:    name,
			ctx:     ctx,
			appName: opt.AppName,
			conf:    conf,
		},
	}
}

func (z *zkKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	opt := utils.ApplyOptions[queryOption](opts...)
	if !opt.withPrefix {
		val, stat, err := z.cli.Get(key)
		return &zkGetValue{key: key, value: string(val), stat: stat, err: err}
	}

	result := &zkGetValue{
		key:   key,
		keys:  []string{key},
		stat:  new(zk.Stat),
		multi: map[string]*zkGetValue{key: {key: key}},
	}
	for i := 0; i < len(result.keys); i++ {
		select {
		case <-ctx.Done():
			result.err = ctx.Err()
			return result
		default:
		}
		next, stat, err := z.cli.Children(result.keys[i])
		if len(result.keys) == 1 {
			result.stat = stat
		}
		if err != nil {
			result.err = err
			return result
		}
		keys := make([]string, 0, len(next))
		for _, desc := range next {
			k := result.keys[i] + constant.Slash + desc
			keys = append(keys, k)
			result.multi[k] = &zkGetValue{key: k}
		}
		result.keys = append(result.keys, keys...)
		if opt.limit > 0 && len(result.keys) > opt.limit {
			break
		}
	}

	if opt.withKeysOnly {
		return result
	}

	for _, descendant := range result.keys {
		// z.cli.Multi() not support get operation
		select {
		case <-ctx.Done():
			result.err = ctx.Err()
			return result
		default:
		}

		val, stat, err := z.cli.Get(descendant)
		if err != nil {
			result.err = err
			return result
		}
		result.multi[descendant] = &zkGetValue{key: descendant, value: string(val), stat: stat, err: err}
	}
	return result
}

func (z *zkKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[writeOption](opts...)
	acls := zk.WorldACL(int32(zk.PermAll))

	var (
		result string
		bs     = []byte(cast.ToString(val))
	)

	// FIXME: exists and then set/create lacks transaction consistency
	exists, stat, err := z.cli.Exists(key)
	if err != nil {
		return &zkPutValue{key: key, stat: stat, err: err}
	}
	version := int32(-1)
	if opt.version > 0 {
		version = int32(opt.version)
	}

	if opt.expired > 0 {
		if exists {
			return &zkPutValue{key: key, stat: stat, err: ErrKeyAlreadyExists}
		}
		result, err = z.cli.CreateTTL(key, bs, zk.FlagTTL, acls, opt.expired)
	} else {
		if exists {
			stat, err = z.cli.Set(key, bs, version)
		} else {
			result, err = z.cli.Create(key, bs, zk.FlagPersistent, acls)
		}
	}
	if err != nil {
		return &zkPutValue{key: key, result: result, stat: stat, err: err}
	}
	if !exists {
		stat = &zk.Stat{Version: 0}
	}
	return &zkPutValue{key: key, result: result, stat: stat, err: err}
}

func (z *zkKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	opt := utils.ApplyOptions[writeOption](opts...)
	version := int32(-1)
	if opt.version > 0 {
		version = int32(opt.version)
	}
	return &zkDelValue{err: z.cli.Delete(key, version)}
}

func (z *zkKV) Exists(ctx context.Context, key string, opts ...utils.OptionExtender) ExistsVal {
	opt := utils.ApplyOptions[queryOption](opts...)
	val, stat, err := z.cli.Exists(key)
	if err != nil || val || !opt.withPrefix {
		return &zkExistsValue{key: key, value: val, stat: stat, err: err}
	}

	keys, stat, err := z.cli.Children(key)
	return &zkExistsValue{key: key, keys: keys, value: len(keys) > 0, stat: stat, err: err}
}

func (z *zkKV) getProxy() any      { return z.cli }
func (z *zkKV) close() (err error) { z.cli.Close(); return }

type zkGetValue struct {
	key, value string
	stat       *zk.Stat
	err        error

	keys  []string
	multi map[string]*zkGetValue
}

func (z *zkGetValue) Err() error {
	if z == nil || z.stat == nil || (z.value == "" && len(z.keys) == 0) {
		return ErrNilValue
	}
	return z.err
}

func (z *zkGetValue) String() string {
	if z == nil {
		return ""
	}
	if len(z.multi) > 0 {
		return z.multi[z.key].String()
	}
	return z.value
}

func (z *zkGetValue) Version() Version {
	if z == nil {
		return newEmptyVersion()
	}
	return &zkVersion{Stat: z.stat}
}

func (z *zkGetValue) KeyValues() KeyValues {
	if z == nil || len(z.multi) == 0 {
		return nil
	}
	kvs := make(KeyValues, 0, len(z.multi))
	for _, k := range z.keys {
		if kv, ok := z.multi[k]; ok {
			kvs = append(kvs, &KeyValue{Key: k, Val: kv.value, Ver: kv.Version()})
		} else {
			kvs = append(kvs, &KeyValue{Key: k, Val: nil, Ver: nil})
		}
	}

	return kvs
}

type zkExistsValue struct {
	key   string
	value bool
	keys  []string
	stat  *zk.Stat
	err   error
}

func (z *zkExistsValue) Bool() bool {
	if z == nil || z.stat == nil {
		return false
	}
	return z.value
}

func (z *zkExistsValue) Err() error {
	if z == nil || z.stat == nil {
		return ErrNilValue
	}
	return z.err
}

func (z *zkExistsValue) Version() Version {
	if z == nil || z.stat == nil {
		return newEmptyVersion()
	}
	return &zkVersion{Stat: z.stat}
}

type zkPutValue struct {
	key    string
	result string
	stat   *zk.Stat
	err    error
}

func (z *zkPutValue) Err() error {
	if z == nil {
		return ErrNilValue
	}
	return z.err
}

func (z *zkPutValue) LeaseID() string {
	if z == nil {
		return ""
	}
	return z.key
}

func (z *zkPutValue) Version() Version {
	if z == nil {
		return newEmptyVersion()
	}
	return &zkVersion{Stat: z.stat}
}

type zkDelValue struct {
	err error
}

func (z *zkDelValue) Err() error {
	if z == nil {
		return ErrNilValue
	}
	return z.err
}

type zkVersion struct {
	*zk.Stat
}

func (z *zkVersion) Version() *big.Int {
	if z == nil || z.Stat == nil {
		return newEmptyVersion().Version()
	}
	return big.NewInt(int64(z.Stat.Version))
}
