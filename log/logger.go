package log

import (
	"context"
	"fmt"
	"log"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/utils/clone"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log/encoder"

	fusCtx "github.com/wfusion/gofusion/context"
)

var (
	globalLogger = defaultLogger(false)

	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]*logger
)

type logger struct {
	name          string
	logger        *zap.Logger
	sugaredLogger *zap.SugaredLogger

	conf   *Conf
	zapCfg *zap.Config
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func Use(name string, opts ...utils.OptionExtender) Loggable {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := appInstances[opt.appName]
	if !ok {
		globalLogger.Debug(context.Background(), "%v [Gofusion] %s instance not found for app: %s",
			syscall.Getpid(), config.ComponentLog, opt.appName, Fields{"component": "log"})
		return globalLogger
	}
	instance, ok := instances[name]
	if !ok {
		instance, ok = instances[config.DefaultInstanceKey]
		if ok {
			instance.Debug(context.Background(), "%v [Gofusion] %s instance not found for name: %s",
				syscall.Getpid(), config.ComponentLog, name, Fields{"component": "log"})
			return instance
		}
		globalLogger.Debug(context.Background(), "%v [Gofusion] %s instance not found for name: %s",
			syscall.Getpid(), config.ComponentLog, name, Fields{"component": "log"})
		return globalLogger
	}

	return instance
}

func defaultLogger(colorful bool) Loggable {
	devCfg := zap.NewDevelopmentConfig()
	devCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	if colorful {
		devCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	devCfg.EncoderConfig.EncodeCaller = encoder.SkipCallerEncoder(encoder.SkipCallers, true)
	zapLogger, _ := devCfg.Build(
		zap.AddStacktrace(zap.PanicLevel),
		zap.AddCaller(),
		zap.Hooks(),
	)
	return &logger{
		zapCfg:        &devCfg,
		logger:        zapLogger,
		sugaredLogger: zapLogger.Sugar(),
	}
}
func (l *logger) Debug(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Debug(msg, fields...)
	} else {
		lg.Debug(ctx, msg, args...)
	}
}
func (l *logger) Info(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Info(msg, fields...)
	} else {
		lg.Info(ctx, msg, args...)
	}
}
func (l *logger) Warn(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Warn(msg, fields...)
	} else {
		lg.Warn(ctx, msg, args...)
	}
}
func (l *logger) Error(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Error(msg, fields...)
	} else {
		lg.Error(ctx, msg, args...)
	}
}
func (l *logger) Panic(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Panic(msg, fields...)
	} else {
		lg.Panic(ctx, msg, args...)
	}
}
func (l *logger) Fatal(ctx context.Context, format string, args ...any) {
	lg, msg, fields := l.sweeten(ctx, format, args...)
	if logger, ok := lg.(*logger); ok {
		logger.logger.Fatal(msg, fields...)
	} else {
		lg.Fatal(ctx, msg, args...)
	}
}
func (l *logger) Level(ctx context.Context) Level {
	ctxLogger := GetCtxLogger(ctx, l)
	lg, ok := ctxLogger.(*logger)
	if !ok {
		return InvalidLevel
	}
	switch lg.logger.Level() {
	case zap.DebugLevel:
		return DebugLevel
	case zap.InfoLevel:
		return InfoLevel
	case zap.WarnLevel:
		return WarnLevel
	case zap.ErrorLevel:
		return ErrorLevel
	case zap.PanicLevel:
		return PanicLevel
	case zap.FatalLevel:
		return FatalLevel
	default:
		return InfoLevel
	}
}

func (l *logger) Config() *outputConf {
	return &outputConf{
		Config:    utils.AnyPtr(*l.conf),
		ZapConfig: clone.Clone(l.zapCfg),
	}
}

func (l *logger) flush() {
	ignore := func(err error) bool {
		// ENOTTY:
		//     ignore sync /dev/stdout: inappropriate ioctl for device errors,
		//     which happens when redirect stderr to stdout
		// EINVAL:
		//     ignore sync /dev/stdout: invalid argument
		for _, target := range []error{syscall.EINVAL, syscall.ENOTTY} {
			if errors.Is(err, target) {
				return true
			}
		}
		return false
	}

	pid := syscall.Getpid()
	if _, err := utils.Catch(l.logger.Sync); err != nil && !ignore(err) {
		log.Printf("%v [Gofusion] %s flush %s logger error: %s", pid, config.ComponentLog, l.name, err)
	}
	if _, err := utils.Catch(l.sugaredLogger.Sync); err != nil && !ignore(err) {
		log.Printf("%v [Gofusion] %s flush %s sugared logger error: %s",
			pid, config.ComponentLog, l.name, err)
	}
}

func (l *logger) sweeten(ctx context.Context, format string, raw ...any) (
	log Loggable, msg string, fields []zap.Field) {
	args := make([]any, 0, len(raw))
	fields = getContextZapFields(ctx)
	for _, arg := range raw {
		switch f := arg.(type) {
		case Fields:
			fields = append(fields, convertFieldsToZapFields(f)...)
			continue
		case zap.Field:
			fields = append(fields, f)
			continue
		}
		args = append(args, arg)
	}

	msg = fmt.Sprintf(format, args...)
	if userID := fusCtx.GetUserID(ctx); utils.IsStrNotBlank(userID) {
		fields = append(fields, zap.String("user_id", userID))
	}
	if traceID := fusCtx.GetTraceID(ctx); utils.IsStrNotBlank(traceID) {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if taskID := fusCtx.GetCronTaskID(ctx); utils.IsStrNotBlank(taskID) {
		fields = append(fields, zap.String("cron_task_id", taskID))
	}
	if taskName := fusCtx.GetCronTaskName(ctx); utils.IsStrNotBlank(taskName) {
		fields = append(fields, zap.String("cron_task_name", taskName))
	}
	if id := utils.GetCtxAny[string](ctx, watermill.ContextKeyMessageUUID); utils.IsStrNotBlank(id) {
		fields = append(fields, zap.String("message_uuid", id))
	}
	if id := utils.GetCtxAny[string](ctx, watermill.ContextKeyRawMessageID); utils.IsStrNotBlank(id) {
		fields = append(fields, zap.String("message_raw_id", id))
	}

	log = GetCtxLogger(ctx, l)
	return
}
