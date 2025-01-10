package customlogger

import (
	"context"
	"reflect"
	"strings"

	"github.com/spf13/cast"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// ZookeeperKVLoggerType FIXME: should not be deleted to avoid compiler optimized
	ZookeeperKVLoggerType = reflect.TypeOf(zookeeperKVLogger{})
	zkKVFields            = log.Fields{"component": strings.ToLower(config.ComponentKV)}
)

type zookeeperKVLogger struct {
	log         log.Loggable
	appName     string
	confName    string
	enabled     bool
	logInstance string
}

func (z *zookeeperKVLogger) Init(log log.Loggable, appName, name, logInstance string) {
	z.log = log
	z.appName = appName
	z.confName = name
	z.logInstance = logInstance
	z.reloadConfig()
}

func (z *zookeeperKVLogger) Printf(format string, args ...any) {
	if z.reloadConfig(); z.enabled {
		ctx, args := z.parseArgs(args...)
		z.logger().Info(ctx, format, args...)
	}
}

func (z *zookeeperKVLogger) logger() log.Loggable {
	if z.log != nil {
		return z.log
	}
	logInstance := config.DefaultInstanceKey
	if z.logInstance != "" {
		logInstance = z.logInstance
	}
	return log.Use(logInstance, log.AppName(z.appName))
}

func (z *zookeeperKVLogger) parseArgs(args ...any) (ctx context.Context, params []any) {
	var ok bool

	if len(args) == 0 {
		return context.Background(), []any{zkKVFields}
	}
	if len(args) == 1 {
		args = append(args, zkKVFields)
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

	params = append(params, zkKVFields)
	return
}

func (z *zookeeperKVLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(z.appName).LoadComponentConfig(config.ComponentKV, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[z.confName]
	if !ok {
		return
	}
	z.enabled = cast.ToBool(cfg["enable_logger"])
	z.logInstance = cast.ToString(cfg["log_instance"])
}
