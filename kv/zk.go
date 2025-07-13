package kv

import (
	"context"
	"math/big"
	"reflect"

	"github.com/dustin/go-humanize"
	"github.com/go-zookeeper/zk"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.uber.org/multierr"

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

func newZKInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storable {
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
			utils.Must(utils.ParseDuration(conf.Endpoint.DialTimeout)),
			zk.WithLogger(reflect.New(inspect.TypeOf(conf.Endpoint.ZooLogger)).Interface().(zk.Logger)),
			zk.WithLogInfo(logInfo),
			zk.WithMaxBufferSize(int(maxBufferSize)),
			zk.WithMaxConnBufferSize(int(connMaxBufferSize)),
		)
	} else {
		conn, ech, err = zk.Connect(
			conf.Endpoint.Addresses,
			utils.Must(utils.ParseDuration(conf.Endpoint.DialTimeout)),
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

func (z *zkKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Got {
	opt := utils.ApplyOptions[option](opts...)
	if opt.withConsistency {
		if _, err := z.cli.Sync(key); err != nil {
			return &zkGetValue{key: key, err: err}
		}
	}

	if !opt.withPrefix {
		if opt.withKeysOnly {
			_, stat, err := z.cli.Exists(key)
			return &zkGetValue{key: key, stat: stat, err: err}
		} else {
			val, stat, err := z.cli.Get(key)
			return &zkGetValue{key: key, value: string(val), stat: stat, err: err}
		}
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
	}

	if opt.withKeysOnly {
		return result
	}

	// FIXME: zookeeper not support pagination
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

func (z *zkKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Put {
	opt := utils.ApplyOptions[option](opts...)
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

func (z *zkKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) Del {
	opt := utils.ApplyOptions[option](opts...)
	version := int32(-1)
	if opt.version > 0 {
		version = int32(opt.version)
	}
	if !opt.withPrefix {
		if err := z.cli.Delete(key, version); err != nil {
			return &zkDelValue{err: err}
		}
		return &zkDelValue{keys: []string{key}}
	}

	result := &zkDelValue{keys: []string{key}}
	for i := 0; i < len(result.keys); i++ {
		select {
		case <-ctx.Done():
			result.err = ctx.Err()
			return result
		default:
		}
		next, _, err := z.cli.Children(result.keys[i])
		if err != nil {
			result.err = err
			return result
		}
		for _, desc := range next {
			result.keys = append(result.keys, result.keys[i]+constant.Slash+desc)
		}
	}
	deleteReqList := make([]any, 0, len(result.keys))
	for i := len(result.keys) - 1; i >= 0; i-- {
		deleteReqList = append(deleteReqList, &zk.DeleteRequest{
			Path:    result.keys[i],
			Version: version,
		})
	}

	multiRsp, err := z.cli.Multi(deleteReqList...)
	if err != nil {
		result.err = err
		return result
	}
	for i := len(multiRsp) - 1; i >= 0; i-- {
		result.err = multierr.Append(result.err, multiRsp[i].Error)
		result.stats = append(result.stats, multiRsp[i].Stat)
	}

	return result
}

func (z *zkKV) Has(ctx context.Context, key string, opts ...utils.OptionExtender) Had {
	opt := utils.ApplyOptions[option](opts...)
	if opt.withConsistency {
		if _, err := z.cli.Sync(key); err != nil {
			return &zkExistsValue{key: key, err: err}
		}
	}

	val, stat, err := z.cli.Exists(key)
	if err != nil || val || !opt.withPrefix {
		return &zkExistsValue{key: key, value: val, stat: stat, err: err}
	}

	keys, stat, err := z.cli.Children(key)
	return &zkExistsValue{key: key, keys: keys, value: len(keys) > 0, stat: stat, err: err}
}

func (z *zkKV) Paginate(ctx context.Context, pattern string, pageSize int, opts ...utils.OptionExtender) Paginated {
	opt := utils.ApplyOptions[option](opts...)
	cursor := 0
	if opt.cursor != nil {
		cursor = cast.ToInt(opt.cursor)
	}
	return &zkPagination{
		abstractPagination: newAbstractPagination(ctx, pageSize, opt),
		kv:                 z,
		first:              true,
		keys:               []string{pattern},
		cursor:             cursor,
	}
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
	if z == nil || z.stat == nil || errors.Is(z.err, zk.ErrNoNode) {
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
	if z == nil {
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

	keys  []string
	stats []*zk.Stat
}

func (z *zkDelValue) Err() error {
	if z == nil {
		return ErrNilValue
	}
	return z.err
}

func (z *zkDelValue) Deleted() []string {
	if z == nil {
		return nil
	}
	return z.keys
}

type zkVersion struct {
	*zk.Stat
}

type zkPagination struct {
	*abstractPagination
	kv *zkKV

	first  bool
	keys   []string
	cursor int
}

func (z *zkPagination) More() bool {
	if z == nil {
		return false
	}
	return z.first || len(z.keys) > z.cursor
}

func (z *zkPagination) Next() (kvs KeyValues, err error) {
	if z == nil {
		return nil, ErrNilValue
	}

	// move the cursor to turn the page for FromKey feature
	var children []string
	if z.first && z.cursor >= len(z.keys) {
		for i := 0; i < len(z.keys) && z.cursor >= len(z.keys); i++ {
			select {
			case <-z.ctx.Done():
				return kvs, z.ctx.Err()
			default:
			}

			parent := z.keys[i]
			if z.opt.withConsistency {
				if _, err = z.kv.cli.Sync(parent); err != nil {
					return
				}
			}

			children, _, err = z.kv.cli.Children(parent)
			if err != nil {
				return nil, err
			}
			for _, child := range children {
				z.keys = append(z.keys, parent+constant.Slash+child)
			}
		}
	}

	z.first = false
	kvs = make(KeyValues, 0, z.count)

	var (
		val  []byte
		stat *zk.Stat
	)
	for cnt := 0; cnt < z.count && z.cursor < len(z.keys); cnt++ {
		select {
		case <-z.ctx.Done():
			return kvs, z.ctx.Err()
		default:
		}

		parent := z.keys[z.cursor]
		if z.opt.withConsistency {
			if _, err = z.kv.cli.Sync(parent); err != nil {
				return
			}
		}

		if z.opt.withKeysOnly {
			kvs = append(kvs, &KeyValue{Key: parent, Val: nil, Ver: new(zkVersion)})
		} else {
			val, stat, err = z.kv.cli.Get(parent)
			if err != nil {
				return
			}
			kvs = append(kvs, &KeyValue{Key: parent, Val: string(val), Ver: &zkVersion{Stat: stat}})
		}
		z.cursor++

		children, _, err = z.kv.cli.Children(parent)
		if err != nil {
			return
		}
		for _, child := range children {
			z.keys = append(z.keys, parent+constant.Slash+child)
		}
	}
	return
}

func (z *zkPagination) Cursor() any {
	if z == nil {
		return nil
	}
	return z.cursor
}

func (z *zkVersion) Version() *big.Int {
	if z == nil || z.Stat == nil {
		return newEmptyVersion().Version()
	}
	return big.NewInt(int64(z.Stat.Version))
}
