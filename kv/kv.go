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
}

type emptyVersion struct{}

func (e *emptyVersion) Version() *big.Int { return big.NewInt(0) }
