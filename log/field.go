package log

import (
	"context"

	"go.uber.org/zap"

	fmkCtx "github.com/wfusion/gofusion/context"
)

type Fields map[string]any

func getContextZapFields(ctx context.Context) (zapFields []zap.Field) {
	v := ctx.Value(fmkCtx.KeyLogFields)
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
