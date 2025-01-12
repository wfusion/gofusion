package kv

import (
	"context"
	"math/big"
	"time"

	"github.com/spf13/cast"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

type etcdKV struct {
	abstractKV

	cli *clientv3.Client
}

func newEtcdInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storage {
	cfg := parseEtcdConfig(ctx, conf, opt)
	cli, err := clientv3.New(*cfg)
	if err != nil {
		panic(err)
	}

	return &etcdKV{
		cli: cli,
		abstractKV: abstractKV{
			name:    name,
			ctx:     ctx,
			appName: opt.AppName,
			conf:    conf,
		},
	}
}

func (e *etcdKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) GetVal {
	opt := utils.ApplyOptions[queryOption](opts...)
	var eopts []clientv3.OpOption
	if opt.withPrefix {
		eopts = append(eopts, clientv3.WithPrefix())
	}
	if opt.withKeysOnly {
		eopts = append(eopts, clientv3.WithKeysOnly())
	}
	if opt.limit > 0 {
		eopts = append(eopts, clientv3.WithLimit(int64(opt.limit)))
	}
	rsp, err := e.cli.Get(ctx, key, eopts...)
	return &etcdGetValue{rsp: rsp, err: err}
}

func (e *etcdKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) PutVal {
	opt := utils.ApplyOptions[writeOption](opts...)

	var (
		leaseID clientv3.LeaseID
		eopts   []clientv3.OpOption
	)
	if opt.leaseID != "" {
		leaseID = clientv3.LeaseID(cast.ToInt64(opt.leaseID))
		eopts = append(eopts, clientv3.WithLease(leaseID))
	} else if opt.expired > 0 {
		rsp, err := clientv3.NewLease(e.cli).Grant(ctx, int64(opt.expired/time.Second))
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
	opt := utils.ApplyOptions[writeOption](opts...)
	rsp, err := e.cli.Delete(ctx, key)
	if err != nil {
		return &etcdDelValue{rsp: rsp, err: err}
	}
	if opt.leaseID != "" {
		rsp, err := clientv3.NewLease(e.cli).Revoke(ctx, clientv3.LeaseID(cast.ToInt64(opt.leaseID)))
		if err != nil {
			return &etcdDelValue{lrsp: rsp, err: err}
		}
	}
	return &etcdDelValue{rsp: rsp, err: err}
}

func (e *etcdKV) getProxy() any { return e.cli }
func (e *etcdKV) close() error  { return e.cli.Close() }

type etcdGetValue struct {
	rsp *clientv3.GetResponse
	err error
}

func (e *etcdGetValue) Err() error {
	if e == nil || e.rsp == nil || len(e.rsp.Kvs) == 0 {
		return ErrNilValue
	}
	return e.err
}

func (e *etcdGetValue) String() string {
	if e == nil || e.rsp == nil || len(e.rsp.Kvs) == 0 {
		return ""
	}
	return string(e.rsp.Kvs[0].Value)
}

func (e *etcdGetValue) Version() Version {
	if e == nil || e.rsp == nil || len(e.rsp.Kvs) == 0 {
		return nil
	}
	return &etcdVersion{KeyValue: e.rsp.Kvs[0]}
}

func (e *etcdGetValue) KeyValues() KeyValues {
	if e == nil || e.rsp == nil || e.rsp.Kvs == nil {
		return nil
	}
	kvs := make(KeyValues, 0, len(e.rsp.Kvs))
	for _, kv := range e.rsp.Kvs {
		kvs = append(kvs, &KeyValue{Key: string(kv.Key), Val: string(kv.Value), Ver: &etcdVersion{KeyValue: kv}})
	}
	return kvs
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
	rsp  *clientv3.DeleteResponse
	lrsp *clientv3.LeaseRevokeResponse
	err  error
}

func (e *etcdDelValue) Err() error {
	if e == nil {
		return ErrNilValue
	}
	return e.err
}

type etcdVersion struct {
	*mvccpb.KeyValue
}

func (e *etcdVersion) Version() *big.Int {
	if e == nil || e.KeyValue == nil {
		return nil
	}
	return big.NewInt(e.KeyValue.Version)
}

func parseEtcdConfig(ctx context.Context, conf *Conf, opt *config.InitOption) (cfg *clientv3.Config) {
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
		RejectOldCluster:     conf.Endpoint.EtcdRejectOldCluster,
		DialOptions:          nil,
		Context:              ctx,
		LogConfig:            nil,
		PermitWithoutStream:  conf.Endpoint.EtcdPermitWithoutStream,
	}

	if conf.EnableLogger {
		if zapCfg := log.Use(conf.LogInstance, log.AppName(opt.AppName)).Config().ZapConfig; zapCfg != nil {
			cfg.LogConfig = zapCfg
		}
	}
	if conf.Endpoint.EtcdAutoSyncInterval != "" {
		cfg.AutoSyncInterval = utils.Must(time.ParseDuration(conf.Endpoint.EtcdAutoSyncInterval))
	}
	if conf.Endpoint.EtcdDialKeepAliveTime != "" {
		cfg.DialKeepAliveTime = utils.Must(time.ParseDuration(conf.Endpoint.EtcdDialKeepAliveTime))
	}
	if conf.Endpoint.EtcdDialKeepAliveTimeout != "" {
		cfg.DialKeepAliveTimeout = utils.Must(time.ParseDuration(conf.Endpoint.EtcdDialKeepAliveTimeout))
	}

	return
}
