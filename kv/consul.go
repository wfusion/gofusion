package kv

import (
	"context"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/cast"
	
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

type consulKV struct {
	abstractKV

	cli *api.Client
}

func newConsulInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) KeyValue {
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
		},
	}
}

func (c *consulKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Value {
	copt := new(api.QueryOptions)
	copt = copt.WithContext(ctx)
	pair, meta, err := c.cli.KV().Get(key, copt)
	return &consulValue{pair: pair, queryMeta: meta, err: err}
}

func (c *consulKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Value {
	opt := utils.ApplyOptions[setOption](opts...)

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
		entry := &api.SessionEntry{
			CreateIndex:   0,
			ID:            key,
			Name:          "",
			Node:          "",
			LockDelay:     0,
			Behavior:      "",
			TTL:           opt.expired.String(),
			Namespace:     "",
			Checks:        nil,
			NodeChecks:    nil,
			ServiceChecks: nil,
		}
		id, meta, err := c.cli.Session().Create(entry, copt)
		if err != nil {
			return &consulValue{pair: nil, writeMeta: meta, err: err}
		}
		pair.Session = id
	}

	meta, err := c.cli.KV().Put(pair, copt)
	return &consulValue{pair: pair, writeMeta: meta, err: err}
}

func (c *consulKV) getProxy() any { return c.cli }
func (c *consulKV) close() error  { return nil }

type consulValue struct {
	pair      *api.KVPair
	queryMeta *api.QueryMeta
	writeMeta *api.WriteMeta
	err       error
}

func (c *consulValue) String() (string, error) {
	if c == nil {
		return "", ErrNilValue
	}
	if c.err != nil {
		return "", c.err
	}
	return string(c.pair.Value), nil
}

func parseConsulConfig(conf *Conf) *api.Config {
	epConf := conf.Endpoint
	cfg := api.DefaultConfig()
	cfg.Address = epConf.Addresses[0]
	cfg.Datacenter = epConf.Datacenter
	if epConf.WaitTime != "" {
		cfg.WaitTime = utils.Must(time.ParseDuration(epConf.WaitTime))
	}

	return cfg
}
