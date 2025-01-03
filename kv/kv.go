package kv

import (
	"context"
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
