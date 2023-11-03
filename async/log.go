package async

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/wfusion/gofusion/config"
)

func (a *asynqConsumer) debug(ctx context.Context, msg string, args ...any) {
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

func (a *asynqConsumer) info(ctx context.Context, msg string, args ...any) {
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

func (a *asynqConsumer) warn(ctx context.Context, msg string, args ...any) {
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

func (a *asynqConsumer) format(src string) (dst string) {
	return fmt.Sprintf("%v [Gofusion] %s %s asynq %s",
		syscall.Getpid(), config.ComponentAsync, a.n, src)
}
