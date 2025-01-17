package kv

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/spf13/cast"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
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

func newEtcdInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) Storable {
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

func (e *etcdKV) Get(ctx context.Context, key string, opts ...utils.OptionExtender) Got {
	opt := utils.ApplyOptions[option](opts...)
	var eopts []clientv3.OpOption
	if opt.withKeysOnly {
		eopts = append(eopts, clientv3.WithKeysOnly())
	}
	if opt.withPrefix {
		eopts = append(eopts, clientv3.WithPrefix())
	}
	if opt.withConsistency {
		ctx = clientv3.WithRequireLeader(ctx)
	}
	rsp, err := e.cli.Get(ctx, key, eopts...)
	return &etcdGetValue{rsp: rsp, err: err}
}

func (e *etcdKV) Put(ctx context.Context, key string, val any, opts ...utils.OptionExtender) Put {
	var (
		leaseID clientv3.LeaseID
		eopts   []clientv3.OpOption
	)

	opt := utils.ApplyOptions[option](opts...)
	if opt.withConsistency {
		ctx = clientv3.WithRequireLeader(ctx)
	}
	if opt.leaseID != "" {
		leaseID = clientv3.LeaseID(cast.ToInt64(opt.leaseID))
		eopts = append(eopts, clientv3.WithLease(leaseID))
	} else if opt.expired > 0 {
		ttl := int64(opt.expired / time.Second)
		if ttl == 0 {
			ttl = 1
		}
		rsp, err := clientv3.NewLease(e.cli).Grant(ctx, ttl)
		if err != nil {
			return &etcdPutValue{err: err}
		}
		leaseID = rsp.ID
		eopts = append(eopts, clientv3.WithLease(leaseID))
	}
	rsp, err := e.cli.Put(ctx, key, cast.ToString(val), eopts...)
	return &etcdPutValue{rsp: rsp, leaseID: leaseID, err: err}
}

func (e *etcdKV) Del(ctx context.Context, key string, opts ...utils.OptionExtender) Del {
	var eopts []clientv3.OpOption
	opt := utils.ApplyOptions[option](opts...)
	if opt.withPrefix {
		eopts = append(eopts, clientv3.WithPrefix())
	}
	if opt.withConsistency {
		ctx = clientv3.WithRequireLeader(ctx)
	}
	rsp, err := e.cli.Delete(ctx, key, eopts...)
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

func (e *etcdKV) Has(ctx context.Context, key string, opts ...utils.OptionExtender) Had {
	ctx = clientv3.WithRequireLeader(ctx)

	opt := utils.ApplyOptions[option](opts...)
	if opt.withConsistency {
		ctx = clientv3.WithRequireLeader(ctx)
	}

	eopts := []clientv3.OpOption{clientv3.WithCountOnly()}
	if opt.withPrefix {
		eopts = append(eopts, clientv3.WithPrefix())
	}
	rsp, err := e.cli.Get(ctx, key, eopts...)
	return &etcdExistsValue{rsp: rsp, err: err}
}

func (e *etcdKV) Paginate(ctx context.Context, pattern string, pageSize int, opts ...utils.OptionExtender) Paginated {
	opt := utils.ApplyOptions[option](opts...)
	eopts := []clientv3.OpOption{
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(int64(pageSize)),
		clientv3.WithRange(clientv3.GetPrefixRangeEnd(pattern)),
	}
	if opt.withKeysOnly {
		eopts = append(eopts, clientv3.WithKeysOnly())
	}
	if opt.withConsistency {
		ctx = clientv3.WithRequireLeader(ctx)
	}
	key := pattern
	if fromKey := cast.ToString(opt.cursor); fromKey != "" {
		key = fromKey + "\x00"
	}
	return &etcdPagination{
		abstractPagination: newAbstractPagination(ctx, pageSize, opt),
		kv:                 e,
		more:               true,
		key:                key,
		opts:               eopts,
	}
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
		return newEmptyVersion()
	}
	return &etcdVersion{KeyValue: e.rsp.Kvs[0], header: e.rsp.Header}
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

type etcdExistsValue struct {
	rsp *clientv3.GetResponse
	err error
}

func (e *etcdExistsValue) Bool() bool {
	if e == nil || e.rsp == nil {
		return false
	}
	return e.rsp.Count > 0
}

func (e *etcdExistsValue) Err() error {
	if e == nil || e.rsp == nil {
		return ErrNilValue
	}
	return e.err
}

func (e *etcdExistsValue) Version() Version {
	if e == nil || e.rsp == nil {
		return newEmptyVersion()
	}
	return &etcdVersion{KeyValue: e.rsp.Kvs[0], header: e.rsp.Header}
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

	header *etcdserverpb.ResponseHeader
}

func (e *etcdVersion) Version() *big.Int {
	if e == nil || e.KeyValue == nil {
		return newEmptyVersion().Version()
	}
	return big.NewInt(e.KeyValue.Version)
}

type etcdPagination struct {
	*abstractPagination
	kv *etcdKV

	more bool
	key  string
	opts []clientv3.OpOption
}

func (e *etcdPagination) More() bool {
	if e == nil {
		return false
	}
	return e.more
}

func (e *etcdPagination) Next() (kvs KeyValues, err error) {
	if e == nil {
		return nil, ErrNilValue
	}

	result, err := e.kv.cli.Get(e.ctx, e.key, e.opts...)
	if err != nil {
		return nil, err
	}

	e.more = result.More
	if len(result.Kvs) > 0 {
		e.key = string(result.Kvs[len(result.Kvs)-1].Key) + "\x00"

		kvs = make(KeyValues, 0, len(result.Kvs))
		for _, kv := range result.Kvs {
			kvs = append(kvs, &KeyValue{
				Key: string(kv.Key),
				Val: kv.Value,
				Ver: &etcdVersion{KeyValue: kv, header: result.Header},
			})
		}
	}

	return
}

func (e *etcdPagination) SetPageSize(pageSize int) {
	if e == nil {
		return
	}

	const limitFuncName = "go.etcd.io/etcd/client/v3.WithLimit"
	for i := 0; i < len(e.opts); i++ {
		name := utils.GetFuncName(e.opts[i])
		if strings.Contains(name, limitFuncName) {
			e.opts = append(e.opts[:i], e.opts[i+1:]...)
			break
		}
	}

	e.abstractPagination.SetPageSize(pageSize)
	e.opts = append(e.opts, clientv3.WithLimit(int64(pageSize)))
}

func (e *etcdPagination) Cursor() any {
	if e == nil {
		return nil
	}
	return e.key
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
