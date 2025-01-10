package kv

import (
	"context"
	"time"

	"github.com/spf13/cast"
	"go.etcd.io/etcd/clientv3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

type etcdKV struct {
	abstractKV

	cli *clientv3.Client
}

func NewEtcdKV(ctx context.Context, name string, conf *Conf, opt *config.InitOption) KeyValue {
	cfg := parseETCDConfig(ctx, conf)
	cli, err := clientv3.New(*cfg)
	if err != nil {
		panic(err)
	}

	return &etcdKV{
		abstractKV: abstractKV{
			name: name,
			ctx:  ctx,
		},
		cli: cli,
	}
}

func (e *etcdKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	//opt := utils.ApplyOptions[getOption](opts...)
	rsp, err := e.cli.Get(ctx, key)
	return &etcdGetValue{rsp: rsp, err: err}
}

func (e *etcdKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[setOption](opts...)

	var (
		leaseID clientv3.LeaseID
		eopts   []clientv3.OpOption
	)
	if opt.expired > 0 {
		lease := clientv3.NewLease(e.cli)
		rsp, err := lease.Grant(ctx, int64(opt.expired/time.Second))
		if err != nil {
			return &etcdPutValue{err: err}
		}
		leaseID = rsp.ID
		eopts = append(eopts, clientv3.WithLease(leaseID))
	}
	rsp, err := e.cli.Put(ctx, key, cast.ToString(val), eopts...)
	return &etcdPutValue{rsp: rsp, leaseID: leaseID, err: err}
}

func (e *etcdKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) DelVal {
	//opt := utils.ApplyOptions[delOption](opts...)
	rsp, err := e.cli.Delete(ctx, key)
	return &etcdDelValue{rsp: rsp, err: err}
}

func (e *etcdKV) getProxy() any { return e.cli }
func (e *etcdKV) close() error  { return e.cli.Close() }

type etcdGetValue struct {
	rsp *clientv3.GetResponse
	err error
}

func (e *etcdGetValue) String() (string, error) {
	if e == nil {
		return "", ErrNilValue
	}
	if e.err != nil {
		return "", e.err
	}

	if len(e.rsp.Kvs) == 0 {
		return "", ErrNilValue
	}

	return string(e.rsp.Kvs[0].Value), nil
}

type etcdPutValue struct {
	leaseID clientv3.LeaseID
	rsp     *clientv3.PutResponse
	err     error
}

func (e *etcdPutValue) LeaseID() string {
	if e == nil {
		return ""
	}
	return cast.ToString(int64(e.leaseID))
}

func (e *etcdPutValue) Err() error {
	if e == nil {
		return ErrNilValue
	}
	return e.err
}

type etcdDelValue struct {
	rsp *clientv3.DeleteResponse
	err error
}

func (e *etcdDelValue) Err() error {
	if e == nil {
		return ErrNilValue
	}
	return e.err
}

func parseETCDConfig(ctx context.Context, conf *Conf) (cfg *clientv3.Config) {
	cfg = &clientv3.Config{
		Endpoints:            conf.Endpoint.Addresses,
		AutoSyncInterval:     0,
		DialTimeout:          utils.Must(time.ParseDuration(conf.Endpoint.DialTimeout)),
		DialKeepAliveTime:    0,
		DialKeepAliveTimeout: 0,
		MaxCallSendMsgSize:   0,
		MaxCallRecvMsgSize:   0,
		TLS:                  nil,
		Username:             conf.Endpoint.User,
		Password:             conf.Endpoint.Password,
		RejectOldCluster:     conf.Endpoint.RejectOldCluster,
		DialOptions:          nil,
		Context:              ctx,
		LogConfig:            nil,
		PermitWithoutStream:  conf.Endpoint.PermitWithoutStream,
	}

	if conf.EnableLogger {
		if zapCfg := log.Use(conf.LogInstance).Config().ZapConfig; zapCfg != nil {
			cfg.LogConfig = zapCfg
		}
	}
	if conf.Endpoint.AutoSyncInterval != "" {
		cfg.AutoSyncInterval = utils.Must(time.ParseDuration(conf.Endpoint.AutoSyncInterval))
	}
	if conf.Endpoint.DialKeepAliveTime != "" {
		cfg.DialKeepAliveTime = utils.Must(time.ParseDuration(conf.Endpoint.DialKeepAliveTime))
	}
	if conf.Endpoint.DialKeepAliveTimeout != "" {
		cfg.DialKeepAliveTimeout = utils.Must(time.ParseDuration(conf.Endpoint.DialKeepAliveTimeout))
	}

	return
}
