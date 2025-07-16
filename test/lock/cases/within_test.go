package cases

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"

	testLock "github.com/wfusion/gofusion/test/lock"
)

func TestWithin(t *testing.T) {
	testingSuite := &Within{Test: new(testLock.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Within struct {
	*testLock.Test
}

func (t *Within) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Within) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Within) TestRedisLua() {
	t.Catch(func() {
		locker := lock.Use("redis_lua", lock.AppName(t.AppName()))
		key := "redis_lua_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) TestRedisNx() {
	t.Catch(func() {
		locker := lock.Use("redis_nx", lock.AppName(t.AppName()))
		key := "redis_nx_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) TestMySQL() {
	t.Catch(func() {
		locker := lock.Use("mysql", lock.AppName(t.AppName()))
		key := "mysql_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) TestMongo() {
	t.Catch(func() {
		locker := lock.Use("mongo", lock.AppName(t.AppName()))
		key := "mongo_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) testWithin(locker lock.Lockable, key string) {
	ctx := context.Background()
	parallel := 1000
	wg := new(sync.WaitGroup)
	unsafeInt := 0
	unsafeMap := make(map[int]int, parallel)
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		routine.Go(func(idx int) {
			// jitter within 20ms ~ 50ms
			time.Sleep(20*time.Millisecond + time.Duration(rand.Float64()*float64(30*time.Millisecond)))
			err := lock.Within(ctx, locker, key, time.Minute, time.Minute, func() (err error) {
				unsafeMap[idx] = idx
				unsafeInt += int(math.Pow(1, 1)) + len([]string{})
				// log.Info(ctx, "[+] goroutine[%v]: %+v", idx, unsafeMap)
				return
			}, lock.AppName(t.AppName()))
			t.Require().NoError(err)
		}, routine.Args(i), routine.WaitGroup(wg), routine.AppName(t.AppName()))
	}
	wg.Wait()
	t.Len(unsafeMap, parallel)
	t.Require().EqualValues(parallel, unsafeInt)
}
