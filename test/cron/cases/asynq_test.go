package cases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/cron"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/redis"

	testCron "github.com/wfusion/gofusion/test/cron"
)

func TestAsynq(t *testing.T) {
	testingSuite := &Asynq{Test: new(testCron.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Asynq struct {
	*testCron.Test
}

func (t *Asynq) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Asynq) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Asynq) TestMultiCronWithoutLock() {
	t.Catch(func() {
		// Given
		expect := time.Duration(5)
		cnt := atomic.NewInt32(0)
		ctx := context.Background()
		t.cleanByQueue(ctx, "gofusion:cron:default")
		defer t.cleanByQueue(ctx, "gofusion:cron:default")

		r1 := cron.Use(nameDefault, cron.AppName(t.AppName()))
		r1.Handle("test", func(ctx context.Context, task cron.Task) (err error) {
			cnt.Add(1)
			log.Info(ctx, "[1] we get cron task: %s", task.Name())
			return
		})
		r2 := cron.Use(nameDefaultDup, cron.AppName(t.AppName()))
		r2.Handle("test", func(ctx context.Context, task cron.Task) (err error) {
			cnt.Add(1)
			log.Info(ctx, "[2] we get cron task: %s", task.Name())
			return
		})

		// When
		t.NoError(r1.Start())
		t.NoError(r2.Start())
		time.Sleep(expect * time.Second)

		// Then
		t.LessOrEqual(cnt.Load(), int32(expect))
	})
}

func (t *Asynq) TestMultiCronWithLock() {
	t.Catch(func() {
		// Given
		expect := time.Duration(5)
		cnt := atomic.NewInt32(0)
		ctx := context.Background()
		t.cleanByQueue(ctx, "gofusion:cron:with_lock")
		defer t.cleanByQueue(ctx, "gofusion:cron:with_lock")

		r1 := cron.Use(nameWithLock, cron.AppName(t.AppName()))
		r1.Handle("with_args", handleWithArgsFunc(nameWithLock))
		r1.Handle("test", func(ctx context.Context, task cron.Task) (err error) {
			cnt.Add(1)
			log.Info(ctx, "[%s] we get cron task: %s", nameWithLock, task.Name())
			return
		})
		r2 := cron.Use(nameWithLockDup, cron.AppName(t.AppName()))
		r2.Handle("with_args", handleWithArgsFunc(nameWithLockDup))
		r2.Handle("test", func(ctx context.Context, task cron.Task) (err error) {
			cnt.Add(1)
			log.Info(ctx, "[%s] we get cron task: %s", nameWithLockDup, task.Name())
			return
		})

		// When
		t.NoError(r1.Start())
		t.NoError(r2.Start())
		time.Sleep(expect * time.Second)

		// Then
		t.LessOrEqual(cnt.Load(), int32(expect))
	})
}

func (t *Asynq) cleanByQueue(ctx context.Context, queue string) {
	pattern := fmt.Sprintf("asynq:{%s}:*", queue)
	if queue == "" {
		pattern = fmt.Sprintf("asynq:{%s:cron}:*", config.Use(t.AppName()).AppName())
	}

	rdsCli := redis.Use(ctx, "default", redis.AppName(t.AppName()))
	keys, err := rdsCli.Keys(ctx, pattern).Result()
	t.NoError(err)

	if len(keys) > 0 {
		rdsCli.Del(ctx, keys...)
	}
}
