package customlogger

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// GormLoggerType FIXME: should not be deleted to avoid compiler optimized
	GormLoggerType = reflect.TypeOf(gormLogger{})
)

func DefaultMySQLLogger() logger.Interface {
	return &gormLogger{
		enabled:                   true,
		slowThreshold:             200 * time.Millisecond,
		logLevel:                  logger.Silent,
		ignoreRecordNotFoundError: true,
	}
}

type gormLogger struct {
	log                       log.Logable
	appName                   string
	confName                  string
	enabled                   bool
	logLevel                  logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

// LogMode log mode
func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	g.logLevel = level
	return g
}

// Info print info
func (g *gormLogger) Info(ctx context.Context, msg string, data ...any) {
	g.reloadConfig()
	if !g.enabled {
		return
	}
	if g.logLevel >= logger.Info {
		g.log.Info(ctx, msg, data...)
	}
}

// Warn print warn messages
func (g *gormLogger) Warn(ctx context.Context, msg string, data ...any) {
	g.reloadConfig()
	if !g.enabled {
		return
	}
	if g.logLevel >= logger.Warn {
		g.log.Warn(ctx, msg, data...)
	}
}

// Error print error messages
func (g *gormLogger) Error(ctx context.Context, msg string, data ...any) {
	g.reloadConfig()
	if !g.enabled {
		return
	}
	if g.logLevel >= logger.Error {
		g.log.Error(ctx, msg, data...)
	}
}

// Trace print sql message
func (g *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	g.reloadConfig()
	if !g.enabled {
		return
	}
	if g.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && g.logLevel >= logger.Error &&
		(!errors.Is(err, gorm.ErrRecordNotFound) || !g.ignoreRecordNotFoundError):
		sql, rows := fc()
		sql = fmt.Sprintf("err[%%s] %s", g.format(sql))
		if rows == -1 {
			g.log.Info(ctx, sql, err.Error(), log.Fields{"latency": elapsed.Milliseconds()})
		} else {
			g.log.Info(ctx, sql, err.Error(), log.Fields{"rows": rows, "latency": elapsed.Milliseconds()})
		}
	case elapsed > g.slowThreshold && g.slowThreshold != 0 && g.logLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v %s", g.slowThreshold, g.format(sql))
		if rows == -1 {
			g.log.Info(ctx, slowLog, log.Fields{"latency": elapsed.Milliseconds()})
		} else {
			g.log.Info(ctx, slowLog, log.Fields{"rows": rows, "latency": elapsed.Milliseconds()})
		}
	case g.logLevel == logger.Info:
		sql, rows := fc()
		sql = g.format(sql)
		if rows == -1 {
			g.log.Info(ctx, sql, log.Fields{"latency": elapsed.Milliseconds()})
		} else {
			g.log.Info(ctx, sql, log.Fields{"rows": rows, "latency": elapsed.Milliseconds()})
		}
	}
}

func (g *gormLogger) Init(log log.Logable, appName, name string) {
	g.log = log
	g.appName = appName
	g.confName = name
	g.ignoreRecordNotFoundError = true
	g.reloadConfig()
}

func (g *gormLogger) format(sql string) string {
	return strings.ReplaceAll(sql, "%", "%%")
}

func (g *gormLogger) getLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return logger.Info
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return g.logLevel
	}
}

func (g *gormLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(g.appName).LoadComponentConfig(config.ComponentDB, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[g.confName]
	if !ok {
		return
	}
	g.enabled = cast.ToBool(cfg["enable_logger"])
	logConfigObj, ok1 := cfg["logger_config"]
	logCfg, ok2 := logConfigObj.(map[string]any)
	if !ok1 || !ok2 {
		return
	}
	g.logLevel = g.getLogLevel(cast.ToString(logCfg["log_level"]))
	g.slowThreshold, _ = time.ParseDuration(cast.ToString(logCfg["slow_threshold"]))
}
