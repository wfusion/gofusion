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

func newConsulInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storage {
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

func (c *consulKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	opt := utils.ApplyOptions[queryOption](opts...)
	copt := new(api.QueryOptions)
	copt = copt.WithContext(ctx)
	if opt.withPrefix {
		// FIXME: consul not support KeysOnly, Limit
		pairs, meta, err := c.cli.KV().List(key, copt)
		if err != nil {
			return &consulGetValue{multi: pairs, meta: meta, err: err}
		}
	}

	pair, meta, err := c.cli.KV().Get(key, copt)
	return &consulGetValue{pair: pair, meta: meta, err: err}
}

func (c *consulKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[writeOption](opts...)

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
	if opt.expired > 0 {
		if opt.expired < consulMinTTL || opt.expired > consulMaxTTL {
			return &consulPutValue{err: ErrInvalidExpiration}
		}
		entry := &api.SessionEntry{
			CreateIndex:   0,
			ID:            key,
			Name:          key,
			Node:          "",
			LockDelay:     0,
			Behavior:      api.SessionBehaviorRelease,
			TTL:           opt.expired.String(),
			Namespace:     "",
			Checks:        nil,
			NodeChecks:    nil,
			ServiceChecks: nil,
		}
		id, meta, err := c.cli.Session().Create(entry, copt)
		if err != nil {
			return &consulPutValue{sessionID: id, pair: pair, meta: meta, err: err}
		}
		pair.Session = id
	}

	meta, err := c.cli.KV().Put(pair, copt)
	return &consulPutValue{pair: pair, meta: meta, err: err}
}

func (c *consulKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	opt := utils.ApplyOptions[writeOption](opts...)
	copt := new(api.WriteOptions)
	copt = copt.WithContext(ctx)
	meta, err := c.cli.KV().Delete(key, copt)
	if err != nil {
		return &consulDelValue{meta: meta, err: err}
	}
	if opt.leaseID != "" {
		if meta, err := c.cli.Session().Destroy(opt.leaseID, copt); err != nil {
			return &consulDelValue{meta: meta, err: err}
		}
	}
	return &consulDelValue{meta: meta, err: err}
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
	if c == nil || c.pair == nil || len(c.pair.Value) == 0 {
		return ErrNilValue
	}
	return c.err
}

func (c *consulGetValue) String() string {
	if c == nil || c.err != nil || c.pair == nil {
		return ""
	}
	return string(c.pair.Value)
}

func (c *consulGetValue) Version() Version {
	if c == nil || c.err != nil || c.pair == nil {
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

type consulPutValue struct {
	sessionID string
	pair      *api.KVPair
	meta      *api.WriteMeta
	err       error
}

func (c *consulPutValue) LeaseID() string {
	if c == nil {
		return ""
	}
	return c.sessionID
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
	return big.NewInt(0).SetUint64(c.ModifyIndex)
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
