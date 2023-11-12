package log

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/wfusion/gofusion/common/utils"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func Debug(ctx context.Context, format string, args ...any) { globalLogger.Debug(ctx, format, args...) }
func Info(ctx context.Context, format string, args ...any)  { globalLogger.Info(ctx, format, args...) }
func Warn(ctx context.Context, format string, args ...any)  { globalLogger.Warn(ctx, format, args...) }
func Error(ctx context.Context, format string, args ...any) { globalLogger.Error(ctx, format, args...) }
func Panic(ctx context.Context, format string, args ...any) { globalLogger.Panic(ctx, format, args...) }
func Fatal(ctx context.Context, format string, args ...any) { globalLogger.Fatal(ctx, format, args...) }

func TimeElapsed(ctx context.Context, logger Loggable, fn func(), format string, args ...any) {
	now := time.Now()
	defer func() {
		elapsed := time.Since(now).Milliseconds()
		if r := recover(); r != nil {
			panic(r)
		}
		if logger != nil {
			logger.Info(ctx, format, append(args, zap.Any("latency", elapsed))...)
		} else {
			Info(ctx, format, append(args, zap.Any("latency", elapsed))...)
		}
	}()

	fn()
}

func GetContextFields(ctx context.Context) Fields {
	return utils.GetCtxAny(ctx, fmkCtx.KeyLogFields, (Fields)(nil))
}

func SetContextFields(ctx context.Context, fields Fields) context.Context {
	return utils.SetCtxAny(ctx, fmkCtx.KeyLogFields, fields)
}
