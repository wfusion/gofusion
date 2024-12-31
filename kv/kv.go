package kv

import (
	"sync"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]KeyValue
)
