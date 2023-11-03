package redis

import (
	"github.com/redis/go-redis/v9"

	"github.com/wfusion/gofusion/common/utils"
)

func WithHook(hooks []redis.Hook) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.hooks = append(o.hooks, hooks...)
	}
}
