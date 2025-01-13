package kv

import (
	"context"
	"math/big"
	"sync"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]Storage
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
		return nil
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
