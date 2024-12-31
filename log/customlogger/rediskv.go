package customlogger

import (
	"context"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"

	rdsDrv "github.com/redis/go-redis/v9"
)

var (
	// RedisKVLoggerType FIXME: should not be deleted to avoid compiler optimized
	RedisKVLoggerType = reflect.TypeOf(redisKVLogger{})
)

type redisKVLogger struct {
	log                  log.Loggable
	appName              string
	confName             string
	enabled              bool
	unloggableCommandSet *utils.Set[string]
}

func (r *redisKVLogger) Init(log log.Loggable, appName, name string) {
	r.log = log
	r.appName = appName
	r.confName = name
	r.reloadConfig()
}

func (r *redisKVLogger) DialHook(next rdsDrv.DialHook) rdsDrv.DialHook {
	return func(ctx context.Context, network, addr string) (c net.Conn, e error) { return next(ctx, network, addr) }
}

func (r *redisKVLogger) ProcessHook(next rdsDrv.ProcessHook) rdsDrv.ProcessHook {
	return func(ctx context.Context, cmd rdsDrv.Cmder) (err error) {
		if !r.isLoggableCommandSet(cmd.Name()) {
			return next(ctx, cmd)
		}

		begin := time.Now()
		if err = next(ctx, cmd); err != nil {
			r.logger().Warn(ctx, "%s failed [command[%s]]", cmd.FullName(), cmd.String(),
				r.fields(log.Fields{"latency": time.Since(begin).Milliseconds()}))
			return
		}

		r.logger().Info(ctx, "%s succeeded [command[%s]]", cmd.FullName(), cmd.String(),
			r.fields(log.Fields{"latency": time.Since(begin).Milliseconds()}))
		return
	}
}

func (r *redisKVLogger) ProcessPipelineHook(next rdsDrv.ProcessPipelineHook) rdsDrv.ProcessPipelineHook {
	return func(ctx context.Context, cmds []rdsDrv.Cmder) (err error) {
		if !r.isLoggable() {
			return next(ctx, cmds)
		}
		begin := time.Now()
		fullNameSb := new(strings.Builder)
		for _, cmd := range cmds {
			_, _ = fullNameSb.WriteString(cmd.FullName() + " -> ")
		}

		if err = next(ctx, cmds); err != nil {
			r.logger().Warn(ctx, "%s failed", fullNameSb.String(),
				r.fields(log.Fields{"latency": time.Since(begin).Milliseconds()}))
			return
		}

		r.logger().Info(ctx, "%s succeeded", fullNameSb.String(),
			r.fields(log.Fields{"latency": time.Since(begin).Milliseconds()}))
		return
	}
}

func (r *redisKVLogger) logger() log.Loggable {
	if r.log != nil {
		return r.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(r.appName))
}

func (r *redisKVLogger) fields(fields log.Fields) log.Fields {
	return utils.MapMerge(fields, redisFields)
}

func (r *redisKVLogger) isLoggableCommandSet(command string) bool {
	if r.confName == "" {
		return true
	}

	r.reloadConfig()
	if !r.enabled {
		return false
	}
	if r.unloggableCommandSet == nil {
		return true
	}
	return !r.unloggableCommandSet.Contains(command)
}

func (r *redisKVLogger) isLoggable() bool {
	if r.confName == "" {
		return true
	}
	r.reloadConfig()
	return r.enabled
}

func (r *redisKVLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentKV, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[r.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg["enable_logger"])
	r.enabled = enabled

	epConfObj, ok1 := cfg["endpoint"]
	epConf, ok2 := epConfObj.(map[string]any)
	if !ok1 || !ok2 {
		return
	}

	unloggableCommandsObj, ok1 := epConf["unloggable_commands"]
	unloggableCommands, ok2 := unloggableCommandsObj.([]string)
	if !ok1 || !ok2 {
		return
	}
	sets := utils.NewSet(unloggableCommands...)
	r.unloggableCommandSet = sets
}
