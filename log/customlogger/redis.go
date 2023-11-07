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
	// RedisLoggerType FIXME: should not be deleted to avoid compiler optimized
	RedisLoggerType = reflect.TypeOf(redisLogger{})
)

type redisLogger struct {
	log                 log.Logable
	appName             string
	confName            string
	enabled             bool
	unlogableCommandSet *utils.Set[string]
}

func (r *redisLogger) Init(log log.Logable, appName, name string) {
	r.log = log
	r.appName = appName
	r.confName = name
	r.reloadConfig()
}

func (r *redisLogger) DialHook(next rdsDrv.DialHook) rdsDrv.DialHook {
	return func(ctx context.Context, network, addr string) (c net.Conn, e error) { return next(ctx, network, addr) }
}

func (r *redisLogger) ProcessHook(next rdsDrv.ProcessHook) rdsDrv.ProcessHook {
	return func(ctx context.Context, cmd rdsDrv.Cmder) (err error) {
		if !r.isLoggableCommandSet(cmd.Name()) {
			return next(ctx, cmd)
		}

		begin := time.Now()
		if err = next(ctx, cmd); err != nil {
			if r.log != nil {
				r.log.Warn(ctx, "[redis] %s failed [command[%s]]", cmd.FullName(), cmd.String(),
					log.Fields{"latency": time.Since(begin).Milliseconds()})
			} else {
				log.Warn(ctx, "[redis] %s failed [command[%s]]", cmd.FullName(), cmd.String(),
					log.Fields{"latency": time.Since(begin).Milliseconds()})
			}
			return
		}

		if r.log != nil {
			r.log.Info(ctx, "[redis] %s succeeded [command[%s]]", cmd.FullName(), cmd.String(),
				log.Fields{"latency": time.Since(begin).Milliseconds()})
		} else {
			log.Info(ctx, "[redis] %s succeeded [command[%s]]", cmd.FullName(), cmd.String(),
				log.Fields{"latency": time.Since(begin).Milliseconds()})
		}
		return
	}
}

func (r *redisLogger) ProcessPipelineHook(next rdsDrv.ProcessPipelineHook) rdsDrv.ProcessPipelineHook {
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
			if r.log != nil {
				r.log.Warn(ctx, "[redis] %s failed", fullNameSb.String(),
					log.Fields{"latency": time.Since(begin).Milliseconds()})
			} else {
				log.Warn(ctx, "[redis] %s failed", fullNameSb.String(),
					log.Fields{"latency": time.Since(begin).Milliseconds()})
			}
			return
		}

		if r.log != nil {
			r.log.Info(ctx, "[redis] %s succeeded", fullNameSb.String(),
				log.Fields{"latency": time.Since(begin).Milliseconds()})
		} else {
			log.Info(ctx, "[redis] %s succeeded", fullNameSb.String(),
				log.Fields{"latency": time.Since(begin).Milliseconds()})
		}
		return
	}
}

func (r *redisLogger) isLoggableCommandSet(command string) bool {
	r.reloadConfig()
	if !r.enabled {
		return false
	}
	if r.unlogableCommandSet == nil {
		return true
	}
	return !r.unlogableCommandSet.Contains(command)
}

func (r *redisLogger) isLoggable() bool { r.reloadConfig(); return r.enabled }

func (r *redisLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentRedis, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[r.confName]
	if !ok {
		return
	}
	enabled := cast.ToBool(cfg["enable_logger"])
	r.enabled = enabled

	unlogableCommandsObj, ok1 := cfg["unlogable_commands"]
	unlogableCommands, ok2 := unlogableCommandsObj.([]string)
	if !ok1 || !ok2 {
		return
	}
	sets := utils.NewSet(unlogableCommands...)
	r.unlogableCommandSet = sets
}
