package mq

import (
	"context"
	"fmt"
	"syscall"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func logError(ctx context.Context, l watermill.LoggerAdapter, app, n, m string, args ...any) {
	l.Error(formatLogMsg(app, n, m, args...), nil, formatFields(ctx))
}
func logInfo(ctx context.Context, l watermill.LoggerAdapter, app, n, m string, args ...any) {
	l.Info(formatLogMsg(app, n, m, args...), formatFields(ctx))
}
func logDebug(ctx context.Context, l watermill.LoggerAdapter, app, n, m string, args ...any) {
	l.Debug(formatLogMsg(app, n, m, args...), formatFields(ctx))
}
func logTrace(ctx context.Context, l watermill.LoggerAdapter, app, n, m string, args ...any) {
	l.Trace(formatLogMsg(app, n, m, args...), formatFields(ctx))
}

func formatFields(ctx context.Context) (fs watermill.LogFields) {
	fs = make(watermill.LogFields, 4)
	if userID := fmkCtx.GetUserID(ctx); utils.IsStrNotBlank(userID) {
		fs["user_id"] = userID
	}
	if traceID := fmkCtx.GetTraceID(ctx); utils.IsStrNotBlank(traceID) {
		fs["trace_id"] = traceID
	}
	if taskID := fmkCtx.GetCronTaskID(ctx); utils.IsStrNotBlank(taskID) {
		fs["cron_task_id"] = taskID
	}
	if taskName := fmkCtx.GetCronTaskName(ctx); utils.IsStrNotBlank(taskName) {
		fs["cron_task_name"] = taskName
	}
	return
}

func formatLogMsg(app, n, src string, args ...any) (dst string) {
	appName := config.Use(app).AppName()
	return fmt.Sprintf("%v [Gofusion] %s %s %s %s",
		syscall.Getpid(), appName, config.ComponentMessageQueue, n, fmt.Sprintf(src, args...))
}
