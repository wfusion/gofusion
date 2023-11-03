package cases

import (
	"context"
	"reflect"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/cron"
	"github.com/wfusion/gofusion/log"
)

var (
	handleWithCallbackType = reflect.TypeOf(handleWithCallback)
)

const (
	nameDefault     = "default"
	nameDefaultDup  = "default_dup"
	nameWithLock    = "with_lock"
	nameWithLockDup = "with_lock_dup"
)

func handleWithCallback(ctx context.Context, task cron.Task) (err error) {
	log.Info(ctx, "we get task: %s", task.Name())
	return
}

type args struct {
	Msg *string `json:"msg"`
}

func handleWithArgsFunc(id string) func(ctx context.Context, arg *args) error {
	return func(ctx context.Context, arg *args) (err error) {
		log.Info(ctx, "we get %s arg.msg: %s", id, cast.ToString(arg.Msg))
		return
	}
}
