package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wfusion/gofusion/common/utils"
)

var Default Dialect = new(defaultDialect)

type defaultDialect struct{}

func (d *defaultDialect) New(ctx context.Context, option Option, opts ...utils.OptionExtender) (
	r *Redis, err error) {
	if option.Cluster {
		opt := d.parseClusterOption(option)
		d.wrapDurationSetter(option.PoolTimeout, func(du time.Duration) { opt.PoolTimeout = du })
		d.wrapDurationSetter(option.DialTimeout, func(du time.Duration) { opt.DialTimeout = du })
		d.wrapDurationSetter(option.ReadTimeout, func(du time.Duration) { opt.ReadTimeout = du })
		d.wrapDurationSetter(option.WriteTimeout, func(du time.Duration) { opt.WriteTimeout = du })
		d.wrapDurationSetter(option.ConnMaxIdleTime, func(du time.Duration) { opt.ConnMaxIdleTime = du })
		d.wrapDurationSetter(option.ConnMaxLifetime, func(du time.Duration) { opt.ConnMaxLifetime = du })
		d.wrapDurationSetter(option.MinRetryBackoff, func(du time.Duration) { opt.MinRetryBackoff = du })
		d.wrapDurationSetter(option.MaxRetryBackoff, func(du time.Duration) { opt.MaxRetryBackoff = du })

		rdsCli := redis.NewClusterClient(opt)

		// authentication check
		if err = rdsCli.Ping(context.Background()).Err(); err != nil {
			return
		}

		newOpt := utils.ApplyOptions[newOption](opts...)
		for _, hook := range newOpt.hooks {
			rdsCli.AddHook(hook)
		}

		return &Redis{redis: rdsCli}, nil

	} else {
		opt := d.parseOption(option)
		d.wrapDurationSetter(option.PoolTimeout, func(du time.Duration) { opt.PoolTimeout = du })
		d.wrapDurationSetter(option.DialTimeout, func(du time.Duration) { opt.DialTimeout = du })
		d.wrapDurationSetter(option.ReadTimeout, func(du time.Duration) { opt.ReadTimeout = du })
		d.wrapDurationSetter(option.WriteTimeout, func(du time.Duration) { opt.WriteTimeout = du })
		d.wrapDurationSetter(option.ConnMaxIdleTime, func(du time.Duration) { opt.ConnMaxIdleTime = du })
		d.wrapDurationSetter(option.ConnMaxLifetime, func(du time.Duration) { opt.ConnMaxLifetime = du })
		d.wrapDurationSetter(option.MinRetryBackoff, func(du time.Duration) { opt.MinRetryBackoff = du })
		d.wrapDurationSetter(option.MaxRetryBackoff, func(du time.Duration) { opt.MaxRetryBackoff = du })

		rdsCli := redis.NewClient(opt)

		// authentication check
		if err = rdsCli.Ping(context.Background()).Err(); err != nil {
			return
		}

		newOpt := utils.ApplyOptions[newOption](opts...)
		for _, hook := range newOpt.hooks {
			rdsCli.AddHook(hook)
		}

		return &Redis{redis: rdsCli}, nil
	}
}

func (d *defaultDialect) wrapDurationSetter(s string, setter func(du time.Duration)) {
	if utils.IsStrBlank(s) {
		return
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	setter(duration)
}

func (d *defaultDialect) parseOption(option Option) (cfg *redis.Options) {
	return &redis.Options{
		Addr:         option.Endpoints[0],
		Username:     option.User,
		Password:     option.Password,
		MaxRetries:   option.MaxRetries,
		MinIdleConns: option.MinIdleConns,
		MaxIdleConns: option.MaxIdleConns,
		PoolSize:     option.PoolSize,
	}
}

func (d *defaultDialect) parseClusterOption(option Option) (cfg *redis.ClusterOptions) {
	return &redis.ClusterOptions{
		Addrs:        option.Endpoints,
		Username:     option.User,
		Password:     option.Password,
		MaxRetries:   option.MaxRetries,
		MinIdleConns: option.MinIdleConns,
		MaxIdleConns: option.MaxIdleConns,
		PoolSize:     option.PoolSize,
	}
}
