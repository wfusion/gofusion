package customlogger

import (
	"context"
	"reflect"

	"github.com/panjf2000/ants/v2"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// RoutineLoggerType FIXME: should not be deleted to avoid compiler optimized
	RoutineLoggerType = reflect.TypeOf(routineLogger{})
	routineFields     = log.Fields{"component": "routine"}
)

func DefaultRoutineLogger() ants.Logger {
	return &routineLogger{
		enabled: true,
	}
}

type routineLogger struct {
	log     log.Loggable
	appName string
	enabled bool
}

func (r *routineLogger) Init(log log.Loggable, appName string) {
	r.log = log
	r.appName = appName
	r.reloadConfig()
}

func (r *routineLogger) Printf(format string, args ...any) {
	if r.reloadConfig(); r.enabled {
		ctx, args := r.parseArgs(args...)
		r.logger().Info(ctx, format, args...)
	}
}

func (r *routineLogger) logger() log.Loggable {
	if r.log != nil {
		return r.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(r.appName))
}

func (r *routineLogger) parseArgs(args ...any) (ctx context.Context, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), []any{routineFields}
	}
	if len(args) == 1 {
		args = append(args, routineFields)
		return context.Background(), args
	}

	params = args
	ctx, ok = args[0].(context.Context)
	if ok {
		params = args[1:]
	}

	if ctx == nil {
		ctx = context.Background()
	}

	params = append(params, routineFields)
	return
}

func (r *routineLogger) reloadConfig() {
	cfg := make(map[string]any)
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentGoroutinePool, &cfg)

	r.enabled = cast.ToBool(cfg["enable_logger"])
}
