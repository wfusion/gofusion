package cases

import (
	"context"
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
	testingSuite := &Within{Test: testLock.T}
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
		locker := lock.Use("redis_lua", lock.AppName(testLock.Component))
		key := "redis_lua_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) TestRedisNx() {
	t.Catch(func() {
		locker := lock.Use("redis_nx", lock.AppName(testLock.Component))
		key := "redis_nx_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) TestMySQL() {
	t.Catch(func() {
		locker := lock.Use("mysql", lock.AppName(testLock.Component))
		key := "mysql_lock_key"
		t.testWithin(locker, key)
	})
}

func (t *Within) testWithin(locker lock.Lockable, key string) {
	ctx := context.Background()
	parallel := 100
	wg := new(sync.WaitGroup)
	unsafeMap := make(map[int]int, parallel)
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		routine.Go(func(idx int) {
			err := lock.Within(ctx, locker, key, 30*time.Second, 30*time.Second, func() (err error) {
				unsafeMap[idx] = idx
				log.Info(ctx, "[+] goroutine[%v]: %+v", idx, unsafeMap)
				return
			}, lock.AppName(testLock.Component))
			t.NoError(err)
		}, routine.Args(i), routine.WaitGroup(wg), routine.AppName(testLock.Component))
	}
	wg.Wait()
	t.Len(unsafeMap, parallel)
}
