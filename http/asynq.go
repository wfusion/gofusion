package http

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/asynq/asynqmon"
	"github.com/wfusion/gofusion/redis"

	rdsDrv "github.com/redis/go-redis/v9"
)

func initAsynq(ctx context.Context, appName string, r IRouter, confs []*asynqConf) {
	if len(confs) == 0 || r == nil {
		return
	}

	for _, conf := range confs {
		switch conf.InstanceType {
		case instanceTypeRedis:
		default:
			panic(errors.Errorf("unknown asynq instance type: %+v", conf.InstanceType))
		}
		connOpt := &asynqRedisConnOpt{UniversalClient: redis.Use(ctx, conf.Instance, redis.AppName(appName))}
		h := asynqmon.New(asynqmon.Options{
			RootPath:          conf.Path,
			RedisConnOpt:      connOpt,
			PayloadFormatter:  nil,
			ResultFormatter:   nil,
			PrometheusAddress: conf.PrometheusAddress,
			ReadOnly:          conf.Readonly,
		})

		r.Any(h.RootPath()+"/*any", gin.WrapH(h))
	}
}

type asynqRedisConnOpt struct{ rdsDrv.UniversalClient }

func (a *asynqRedisConnOpt) MakeRedisClient() any { return a.UniversalClient }
