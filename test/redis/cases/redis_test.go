package cases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/redis"

	testRedis "github.com/wfusion/gofusion/test/redis"
)

func TestRedis(t *testing.T) {
	testingSuite := &Redis{Test: new(testRedis.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Redis struct {
	*testRedis.Test
}

func (t *Redis) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Redis) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Redis) TestGetSet() {
	t.Catch(func() {
		// Given
		key := "test:getset:key"
		val := "this is a value"
		ctx := context.Background()
		rdsCli := redis.Use(ctx, nameDefault, redis.AppName(t.AppName()))

		// When
		t.Require().NoError(rdsCli.Set(ctx, key, val, time.Second).Err())
		defer rdsCli.Del(ctx, key)

		// Then
		actual, err := rdsCli.Get(ctx, key).Result()
		t.Require().NoError(err)
		t.Require().Equal(val, actual)
	})
}

func (t *Redis) TestSubscribe() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		rdsCli := redis.Use(ctx, nameDefault, redis.AppName(t.AppName()))

		// When
		pubsub := rdsCli.Subscribe(ctx, "asynq:cancel")

		// Then
		_, err := pubsub.ReceiveTimeout(ctx, 500*time.Millisecond)
		t.Require().NoError(err)
	})
}
