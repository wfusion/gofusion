package redis

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/wfusion/gofusion/common/utils"
)

var Default Dialect = new(defaultDialect)

type defaultDialect struct{}

func (d *defaultDialect) New(ctx context.Context, option Option, opts ...utils.OptionExtender) (r *Redis, err error) {
	if len(option.Endpoints) == 0 {
		return nil, errors.New("redis endpoints are empty")
	}

	if option.Cluster {
		opt := d.parseClusterOption(option)
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

func (d *defaultDialect) wrapDurationSetter(s utils.Duration, setter func(du time.Duration)) {
	setter(s.Duration)
}

func (d *defaultDialect) parseOption(option Option) (cfg *redis.Options) {
	return &redis.Options{
		Addr:            option.Endpoints[0],
		Username:        option.User,
		Password:        option.Password,
		MaxRetries:      option.MaxRetries,
		MinIdleConns:    option.MinIdleConns,
		MaxIdleConns:    option.MaxIdleConns,
		PoolSize:        option.PoolSize,
		PoolTimeout:     option.PoolTimeout.Duration,
		DialTimeout:     option.DialTimeout.Duration,
		ReadTimeout:     option.ReadTimeout.Duration,
		WriteTimeout:    option.WriteTimeout.Duration,
		ConnMaxIdleTime: option.ConnMaxIdleTime.Duration,
		ConnMaxLifetime: option.ConnMaxLifetime.Duration,
		MinRetryBackoff: option.MinRetryBackoff.Duration,
		MaxRetryBackoff: option.MaxRetryBackoff.Duration,
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
