package kv

import (
	"context"
	"math/big"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

const (
	consulMinTTL = 10 * time.Second
	consulMaxTTL = 24 * time.Hour
)

type consulKV struct {
	abstractKV

	cli *api.Client
}

func newConsulInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storable {
	copt := parseConsulConfig(conf)
	cli, err := api.NewClient(copt)
	if err != nil {
		panic(err)
	}

	return &consulKV{
		cli: cli,
		abstractKV: abstractKV{
			name:    name,
			ctx:     ctx,
			appName: opt.AppName,
			conf:    conf,
		},
	}
}

func (c *consulKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Got {
	opt := utils.ApplyOptions[option](opts...)
	copt := new(api.QueryOptions)
	copt = copt.WithContext(ctx)
	if opt.withConsistency {
		copt.RequireConsistent = true
	}

	if !opt.withPrefix {
		// FIXME: consul not support exists or keys only
		pair, meta, err := c.cli.KV().Get(key, copt)
		if opt.withKeysOnly {
			pair.Value = nil
		}
		return &consulGetValue{pair: pair, meta: meta, err: err}
	}

	if opt.withKeysOnly {
		keys, meta, err := c.cli.KV().Keys(key, "", copt)
		pairs := make(api.KVPairs, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, &api.KVPair{
				Key:         k,
				CreateIndex: 0,
				ModifyIndex: 0,
				LockIndex:   0,
				Flags:       0,
				Value:       nil,
				Session:     "",
				Namespace:   "",
				Partition:   "",
			})
		}
		return &consulGetValue{multi: pairs, meta: meta, err: err}
	}

	pairs, meta, err := c.cli.KV().List(key, copt)
	return &consulGetValue{multi: pairs, meta: meta, err: err}
}

func (c *consulKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Put {
	opt := utils.ApplyOptions[option](opts...)

	copt := new(api.WriteOptions)
	copt = copt.WithContext(ctx)
	pair := &api.KVPair{
		Key:         key,
		CreateIndex: 0,
		ModifyIndex: 0,
		LockIndex:   0,
		Flags:       0,
		Value:       []byte(cast.ToString(val)),
		Session:     "",
		Namespace:   "",
		Partition:   "",
	}
	if opt.expired <= 0 {
		meta, err := c.cli.KV().Put(pair, copt)
		return &consulPutValue{pair: pair, meta: meta, err: err}
	}

	if opt.expired < consulMinTTL || opt.expired > consulMaxTTL {
		return &consulPutValue{err: ErrInvalidExpiration}
	}
	entry := &api.SessionEntry{
		CreateIndex:   0,
		ID:            key,
		Name:          key,
		Node:          "",
		LockDelay:     0,
		Behavior:      api.SessionBehaviorDelete,
		TTL:           opt.expired.String(),
		Namespace:     "",
		Checks:        nil,
		NodeChecks:    nil,
		ServiceChecks: nil,
	}
	id, meta, err := c.cli.Session().CreateNoChecks(entry, copt)
	if err != nil {
		return &consulPutValue{pair: pair, meta: meta, err: err}
	}
	pair.Session = id
	ok, meta, err := c.cli.KV().Acquire(pair, copt)
	if !ok {
		return &consulPutValue{pair: pair, meta: meta, err: ErrKeyAlreadyExists}
	}
	return &consulPutValue{pair: pair, meta: meta, err: err}
}

func (c *consulKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) (val Del) {
	opt := utils.ApplyOptions[option](opts...)
	copt := new(api.WriteOptions)
	copt = copt.WithContext(ctx)
	if opt.leaseID != "" {
		meta, err := c.cli.Session().Destroy(opt.leaseID, copt)
		return &consulDelValue{meta: meta, err: err}
	}
	if opt.withPrefix {
		meta, err := c.cli.KV().DeleteTree(key, copt)
		return &consulDelValue{meta: meta, err: err}
	}
	meta, err := c.cli.KV().Delete(key, copt)
	return &consulDelValue{meta: meta, err: err}
}

func (c *consulKV) Has(ctx context.Context, key string, opts ...utils.OptionExtender) Had {
	opt := utils.ApplyOptions[option](opts...)
	copt := new(api.QueryOptions)
	copt = copt.WithContext(ctx)
	if opt.withConsistency {
		copt.RequireConsistent = true
	}
	if !opt.withPrefix {
		pair, meta, err := c.cli.KV().Get(key, copt)
		return &consulExistsValue{key: key, pair: pair, meta: meta, err: err}
	}

	keys, meta, err := c.cli.KV().Keys(key, "", copt)
	if err != nil {
		return &consulExistsValue{key: key, keys: keys, meta: meta, err: err}
	}
	return &consulExistsValue{key: key, keys: keys, meta: meta}
}

func (c *consulKV) Paginate(ctx context.Context, pattern string, pageSize int, opts ...utils.OptionExtender) Paginated {
	opt := utils.ApplyOptions[option](opts...)
	copt := new(api.QueryOptions)
	copt = copt.WithContext(ctx)
	if opt.withConsistency {
		copt.RequireConsistent = true
	}
	keys, _, err := c.cli.KV().Keys(pattern, "", copt)
	return &consulPagination{
		abstractPagination: newAbstractPagination(ctx, pageSize, opt),
		kv:                 c,
		opts:               copt,
		prefix:             pattern,
		keys:               keys,
		cursor:             0,
		err:                err,
	}
}

func (c *consulKV) getProxy() any { return c.cli }
func (c *consulKV) close() error  { return nil }

type consulGetValue struct {
	pair *api.KVPair
	meta *api.QueryMeta
	err  error

	multi api.KVPairs
}

func (c *consulGetValue) Err() error {
	if c == nil || (c.pair == nil && c.multi == nil) {
		return ErrNilValue
	}
	return c.err
}

func (c *consulGetValue) String() string {
	if c == nil || (c.pair == nil && c.multi == nil) {
		return ""
	}
	if len(c.multi) > 0 {
		return string(c.multi[0].Value)
	}
	return string(c.pair.Value)
}

func (c *consulGetValue) Version() Version {
	if c == nil || c.pair == nil {
		return newEmptyVersion()
	}
	return &consulVersion{KVPair: c.pair}
}

func (c *consulGetValue) KeyValues() KeyValues {
	if c == nil || c.multi == nil {
		return nil
	}
	kvs := make(KeyValues, 0, len(c.multi))
	for _, kv := range c.multi {
		kvs = append(kvs, &KeyValue{Key: kv.Key, Val: string(kv.Value), Ver: &consulVersion{KVPair: kv}})
	}
	return kvs
}

type consulExistsValue struct {
	pair *api.KVPair
	meta *api.QueryMeta
	key  string
	keys []string
	err  error
}

func (c *consulExistsValue) Bool() bool {
	if c == nil || c.err != nil || (c.pair == nil && len(c.keys) == 0) {
		return false
	}
	return true
}

func (c *consulExistsValue) Err() error {
	if c == nil || (c.pair == nil && len(c.keys) == 0) {
		return ErrNilValue
	}
	return c.err
}

func (c *consulExistsValue) Version() Version {
	if c == nil || c.pair == nil {
		return newEmptyVersion()
	}
	return &consulVersion{KVPair: c.pair}
}

type consulPutValue struct {
	pair *api.KVPair
	meta *api.WriteMeta
	err  error
}

func (c *consulPutValue) LeaseID() string {
	if c == nil || c.pair == nil {
		return ""
	}
	return c.pair.Session
}

func (c *consulPutValue) Err() error {
	if c == nil {
		return ErrNilValue
	}
	return c.err
}

type consulDelValue struct {
	meta *api.WriteMeta
	err  error
}

func (c *consulDelValue) Err() error {
	if c == nil {
		return ErrNilValue
	}
	return c.err
}

type consulVersion struct {
	*api.KVPair
}

func (c *consulVersion) Version() *big.Int {
	if c == nil || c.KVPair == nil {
		return newEmptyVersion().Version()
	}
	return big.NewInt(0).SetUint64(c.KVPair.ModifyIndex)
}

type consulPagination struct {
	*abstractPagination
	kv   *consulKV
	opts *api.QueryOptions

	err    error
	prefix string
	keys   []string
	cursor int
}

func (c *consulPagination) More() bool {
	if c == nil || c.err != nil {
		return false
	}
	return len(c.keys) > c.cursor
}

func (c *consulPagination) Next() (kvs KeyValues, err error) {
	if c == nil {
		return nil, ErrNilValue
	}
	if c.err != nil {
		return nil, c.err
	}

	var pair *api.KVPair

	kvs = make(KeyValues, 0, c.count)
	for cnt := 0; cnt < c.count && c.cursor < len(c.keys); cnt++ {
		select {
		case <-c.ctx.Done():
			return kvs, c.ctx.Err()
		default:
		}

		key := c.keys[c.cursor]
		c.cursor++

		if c.opt.withKeysOnly {
			kvs = append(kvs, &KeyValue{Key: key, Ver: new(consulVersion)})
			continue
		}

		pair, _, err = c.kv.cli.KV().Get(key, c.opts)
		if err != nil {
			return
		}
		if pair == nil {
			continue
		}
		kvs = append(kvs, &KeyValue{Key: key, Val: string(pair.Value), Ver: &consulVersion{KVPair: pair}})
	}
	return
}

func parseConsulConfig(conf *Conf) *api.Config {
	epConf := conf.Endpoint
	cfg := api.DefaultConfig()
	cfg.Address = epConf.Addresses[0]
	cfg.Datacenter = epConf.ConsulDatacenter
	if epConf.ConsulWaitTime != "" {
		cfg.WaitTime = utils.Must(time.ParseDuration(epConf.ConsulWaitTime))
	}

	return cfg
}
