package trace

import (
	"sync"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]TracerProvider
)
