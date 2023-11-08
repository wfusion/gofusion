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
	log     log.Logable
	appName string
	enabled bool
}

func (r *routineLogger) Init(log log.Logable, appName string) {
	r.log = log
	r.appName = appName
	r.reloadConfig()
}

func (r *routineLogger) Printf(format string, args ...any) {
	if r.reloadConfig(); r.enabled {
		r.logger().Info(context.Background(), format, append(args, routineFields)...)
	}
}

func (r *routineLogger) logger() log.Logable {
	if r.log != nil {
		return r.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(r.appName))
}

func (r *routineLogger) reloadConfig() {
	cfg := make(map[string]any)
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentGoroutinePool, &cfg)

	r.enabled = cast.ToBool(cfg["enabled_logger"])
}
