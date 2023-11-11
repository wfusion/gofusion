package cases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"

	testLock "github.com/wfusion/gofusion/test/lock"
)

func TestExpired(t *testing.T) {
	testingSuite := &Expired{Test: new(testLock.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Expired struct {
	*testLock.Test
}

func (t *Expired) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Expired) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Expired) TestRedisLua() {
	t.Catch(func() {
		locker := lock.Use("redis_lua", lock.AppName(t.AppName()))
		key := "redis_lua_lock_expired_key"
		t.testExpired(locker, key, 100*time.Millisecond, time.Second)
	})
}

func (t *Expired) TestRedisNx() {
	t.Catch(func() {
		locker := lock.Use("redis_nx", lock.AppName(t.AppName()))
		key := "redis_nx_lock_expired_key"
		t.testExpired(locker, key, 100*time.Millisecond, 500*time.Millisecond)
	})
}

func (t *Expired) TestMySQL() {
	t.Catch(func() {
		locker := lock.Use("mysql", lock.AppName(t.AppName()))
		key := "mysql_lock_expired_key"
		t.testExpired(locker, key, 100*time.Millisecond, 500*time.Millisecond)
	})
}

func (t *Expired) TestMongo() {
	t.Catch(func() {
		locker := lock.Use("mongo", lock.AppName(t.AppName()))
		key := "mongo_lock_expired_key"
		t.testExpired(locker, key, 100*time.Millisecond, 2*time.Second)
	})
}

func (t *Expired) testExpired(locker lock.Lockable, key string, expired time.Duration, waitTime time.Duration) {
	t.Catch(func() {
		ctx := context.Background()
		for i := 0; i < 5; i++ {
			err := locker.Lock(ctx, key, lock.Expire(expired))
			if err != nil {
				t.FailNow("try to lock failed", err)
			}
			time.Sleep(waitTime)
		}
	})
}
