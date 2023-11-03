package redis

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/drivers/redis"
	"github.com/wfusion/gofusion/common/utils"

	rdsDrv "github.com/redis/go-redis/v9"
)

var (
	rwlock    = new(sync.RWMutex)
	instances map[string]map[string]*instance
)

type instance struct {
	name  string
	redis *redis.Redis
}

func (i *instance) GetProxy() rdsDrv.UniversalClient {
	return i.redis.GetProxy()
}

type Redis struct {
	rdsDrv.UniversalClient
	Name string
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func Use(ctx context.Context, name string, opts ...utils.OptionExtender) rdsDrv.UniversalClient {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := instances[opt.appName]
	if !ok {
		panic(errors.Errorf("redis instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("redis instance not found for name: %s", name))
	}
	return &Redis{UniversalClient: instance, Name: name}
}
