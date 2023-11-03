package context

import (
	"context"

	"github.com/wfusion/gofusion/common/utils"
)

func GetUserID(ctx context.Context, args ...string) (userID string) {
	return utils.GetCtxAny(ctx, KeyUserID, args...)
}

func SetUserID(ctx context.Context, val string) context.Context {
	return utils.SetCtxAny(ctx, KeyUserID, val)
}

func GetTraceID(ctx context.Context, args ...string) (traceID string) {
	return utils.GetCtxAny(ctx, KeyTraceID, args...)
}

func SetTraceID(ctx context.Context, val string) context.Context {
	return utils.SetCtxAny(ctx, KeyTraceID, val)
}

func GetLangs(ctx context.Context, args ...[]string) (langs []string) {
	return utils.GetCtxAny(ctx, KeyLangs, args...)
}

func SetLangs(ctx context.Context, val []string) context.Context {
	return utils.SetCtxAny(ctx, KeyLangs, val)
}

func GetCronTaskID(ctx context.Context, args ...string) (userID string) {
	return utils.GetCtxAny(ctx, KeyCronTaskID, args...)
}

func SetCronTaskID(ctx context.Context, val string) context.Context {
	return utils.SetCtxAny(ctx, KeyCronTaskID, val)
}

func GetCronTaskName(ctx context.Context, args ...string) (userID string) {
	return utils.GetCtxAny(ctx, KeyCronTaskName, args...)
}

func SetCronTaskName(ctx context.Context, val string) context.Context {
	return utils.SetCtxAny(ctx, KeyCronTaskName, val)
}
