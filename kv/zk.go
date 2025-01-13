package kv

import (
	"context"
	"math/big"
	"reflect"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-zookeeper/zk"
	"github.com/spf13/cast"

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
		logInfo := log.Use(conf.LogInstance, log.AppName(opt.AppName)).Level(ctx) > log.InfoLevel
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
	for _, descendant := range result.keys {
		select {
		case <-ctx.Done():
			result.err = ctx.Err()
			return result
		default:
		}
		next, stat, err := z.cli.Children(descendant)
		if len(result.keys) == 1 {
			result.stat = stat
		}
		if err != nil {
			result.err = err
			return result
		}
		for _, desc := range next {
			result.multi[desc] = &zkGetValue{key: desc}
		}
		result.keys = append(result.keys, next...)
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
			result.stat = stat
			return result
		}
		result.multi[descendant] = &zkGetValue{key: descendant, value: string(val), stat: stat, err: err}
	}
	return result
}

func (z *zkKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[setOption](opts...)
	acls := zk.WorldACL(int32(zk.PermAll))

	var (
		err    error
		result string
		bs     = []byte(cast.ToString(val))
	)
	if opt.expired > 0 {
		result, err = z.cli.CreateTTL(key, bs, int32(zk.FlagTTL), acls, opt.expired)
	} else {
		result, err = z.cli.Create(key, bs, zk.FlagPersistent, acls)
	}
	return &zkPutValue{key: key, result: result, err: err}
}

func (z *zkKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	opt := utils.ApplyOptions[delOption](opts...)
	return &zkDelValue{err: z.cli.Delete(key, int32(opt.version))}
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
	if z == nil || z.stat == nil {
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

type zkPutValue struct {
	key    string
	result string
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
	return big.NewInt(int64(z.Stat.Version))
}
