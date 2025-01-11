package kv

import (
	"context"
	"math/big"
	"sync"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]KeyValue
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

type emptyVersion struct{}

func (e *emptyVersion) Version() *big.Int { return big.NewInt(0) }
