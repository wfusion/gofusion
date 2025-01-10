package kv

import (
	"context"
	"math/big"
	"reflect"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-zookeeper/zk"
	"github.com/spf13/cast"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/log"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

type zkKV struct {
	abstractKV

	cli *zk.Conn
	ech <-chan zk.Event
}

func newZKInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) KeyValue {
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
		cli: conn,
		ech: ech,
		abstractKV: abstractKV{
			name:    name,
			ctx:     ctx,
			appName: opt.AppName,
		},
	}
}

func (z *zkKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	//opt := utils.ApplyOptions[getOption](opts...)
	val, stat, err := z.cli.Get(key)
	return &zkGetValue{stat: stat, value: string(val), err: err}
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
	stat  *zk.Stat
	value string
	err   error
}

func (z *zkGetValue) String() (string, error) {
	if z == nil {
		return "", ErrNilValue
	}
	return z.value, z.err
}

func (z *zkGetValue) Version() (Version, error) {
	if z == nil {
		return nil, ErrNilValue
	}
	return &zkVersion{Stat: z.stat}, nil
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
