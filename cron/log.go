package cron

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/wfusion/gofusion/config"
)

func (a *asynqRouter) debug(ctx context.Context, msg string, args ...any) {
	msg = a.format(msg)
	if a.logger == nil {
		log.Printf(msg, args...)
	} else {
		logArgs := make([]any, 0, len(args)+2)
		logArgs = append(logArgs, ctx, msg)
		logArgs = append(logArgs, args...)
		a.logger.Debug(logArgs...)
	}
}

func (a *asynqRouter) info(ctx context.Context, msg string, args ...any) {
	msg = a.format(msg)
	if a.logger == nil {
		log.Printf(msg, args...)
	} else {
		logArgs := make([]any, 0, len(args)+2)
		logArgs = append(logArgs, ctx, msg)
		logArgs = append(logArgs, args...)
		a.logger.Info(logArgs...)
	}
}

func (a *asynqRouter) warn(ctx context.Context, msg string, args ...any) {
	msg = a.format(msg)
	if a.logger == nil {
		log.Printf(msg, args...)
	} else {
		logArgs := make([]any, 0, len(args)+2)
		logArgs = append(logArgs, ctx, msg)
		logArgs = append(logArgs, args...)
		a.logger.Warn(logArgs...)
	}
}

func (a *asynqRouter) format(src string) (dst string) {
	return fmt.Sprintf("%v [Gofusion] %s %s asynq(%s) %s",
		syscall.Getpid(), config.ComponentCron, a.n, a.id, src)
}
