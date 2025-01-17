package kv

import (
	"context"
	"math/big"
	"sync"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]Storable
)

type abstractKV struct {
	ctx context.Context

	appName string
	name    string
	conf    *Conf
}

func (a *abstractKV) config() *Conf {
	if a == nil {
		return nil
	}
	return a.conf
}

type abstractPagination struct {
	ctx   context.Context
	opt   *option
	count int
}

func newAbstractPagination(ctx context.Context, count int, opt *option) *abstractPagination {
	return &abstractPagination{
		ctx:   ctx,
		opt:   opt,
		count: count,
	}
}

func (a *abstractPagination) More() bool {
	panic(ErrNotImplement)
}

func (a *abstractPagination) Next() (KeyValues, error) {
	panic(ErrNotImplement)
}

func (a *abstractPagination) SetPageSize(pageSize int) {
	a.count = pageSize
}

type emptyVersion struct {
	existKV bool
}

func newEmptyVersion() *emptyVersion {
	return &emptyVersion{existKV: false}
}

func newDefaultVersion() *emptyVersion {
	return &emptyVersion{existKV: true}
}

func (e *emptyVersion) Version() *big.Int {
	if !e.existKV {
		return big.NewInt(-1)
	}
	return big.NewInt(0)
}

type KeyValue struct {
	Key string
	Val any
	Ver Version
}

type KeyValues []*KeyValue

func (k KeyValues) Map() map[string]any {
	if k == nil {
		return make(map[string]any)
	}
	m := make(map[string]any, len(k))
	for _, kv := range k {
		m[kv.Key] = kv.Val
	}
	return m
}

func (k KeyValues) Keys() []string {
	if k == nil {
		return nil
	}
	keys := make([]string, 0, len(k))
	for _, kv := range k {
		keys = append(keys, kv.Key)
	}
	return keys
}

func (k KeyValues) Values() []any {
	if k == nil {
		return nil
	}
	values := make([]any, 0, len(k))
	for _, kv := range k {
		values = append(values, kv.Val)
	}
	return values
}

func (k KeyValues) VersionMap() map[string]Version {
	if k == nil {
		return make(map[string]Version)
	}
	m := make(map[string]Version, len(k))
	for _, kv := range k {
		m[kv.Key] = kv.Ver
	}
	return m
}
