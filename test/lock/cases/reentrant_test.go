package cases

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"

	testLock "github.com/wfusion/gofusion/test/lock"
)

func TestReentrant(t *testing.T) {
	testingSuite := &Reentrant{Test: new(testLock.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Reentrant struct {
	*testLock.Test
}

func (t *Reentrant) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Reentrant) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Reentrant) TestRedisLua() {
	t.Catch(func() {
		locker := lock.Use("redis_lua", lock.AppName(t.AppName()))
		key := "redis_lua_lock_reentrant_key"
		t.testReentrant(locker, key)
	})
}

func (t *Reentrant) TestMongo() {
	t.Catch(func() {
		locker := lock.Use("mongo", lock.AppName(t.AppName()))
		key := "mongo_lock_reentrant_key"
		t.testReentrant(locker, key)
	})
}

func (t *Reentrant) testReentrant(locker lock.Lockable, key string) {
	ctx := context.Background()
	parallel := 100
	wg := new(sync.WaitGroup)
	unsafeInt := 0
	unsafeMap := make(map[int]int, parallel)
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		routine.Go(func(idx int) {
			// jitter within 20ms ~ 50ms
			reentrantKey := utils.ULID()
			time.Sleep(20*time.Millisecond + time.Duration(rand.Float64()*float64(30*time.Millisecond)))
			err := lock.Within(ctx, locker, key, time.Minute, time.Minute, func() (err error) {
				unsafeMap[idx] = idx
				unsafeInt += int(math.Pow(1, 1)) + len([]string{})

				rwg := new(sync.WaitGroup)
				for j := 0; j < parallel; j++ {
					rwg.Add(1)
					routine.Go(func() {
						t.Require().NoError(locker.Lock(ctx, key, lock.Expire(time.Millisecond), lock.ReentrantKey(reentrantKey)))
						defer t.Require().NoError(locker.Unlock(ctx, key, lock.ReentrantKey(reentrantKey)))
						// jitter within 10ms
						time.Sleep(time.Duration(rand.Float64() * float64(10*time.Millisecond)))
					}, routine.WaitGroup(rwg), routine.AppName(t.AppName()))
				}
				rwg.Wait()

				return
			}, lock.ReentrantKey(reentrantKey), lock.AppName(t.AppName()))
			t.Require().NoError(err)
		}, routine.Args(i), routine.WaitGroup(wg), routine.AppName(t.AppName()))
	}

	wg.Wait()
	t.Len(unsafeMap, parallel)
	t.Require().EqualValues(parallel, unsafeInt)
}
