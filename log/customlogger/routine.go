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
		ctx := context.Background()
		r.log.Info(ctx, format, args...)
	}
}

func (r *routineLogger) reloadConfig() {
	cfg := make(map[string]any)
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentGoroutinePool, &cfg)

	r.enabled = cast.ToBool(cfg["enabled_logger"])
}
