package log

import (
	"context"

	"go.uber.org/zap"
	
	"github.com/wfusion/gofusion/common/utils"

	fusCtx "github.com/wfusion/gofusion/context"
)

type Fields map[string]any

func getContextZapFields(ctx context.Context) (zapFields []zap.Field) {
	v := ctx.Value(fusCtx.KeyLogFields)
	if v == nil {
		return
	}
	field, ok := v.(Fields)
	if !ok {
		return
	}
	return convertFieldsToZapFields(field)
}

func convertFieldsToZapFields(fields Fields) (zapFields []zap.Field) {
	zapFields = make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return
}

func GetCtxLogger(ctx context.Context, args ...Loggable) (logger Loggable) {
	return utils.GetCtxAny(ctx, fusCtx.KeyLoggable, args...)
}

func SetCtxLogger(ctx context.Context, val Loggable) context.Context {
	return utils.SetCtxAny(ctx, fusCtx.KeyLoggable, val)
}
