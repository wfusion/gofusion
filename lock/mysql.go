package lock

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/routine"
)

const (
	mysqlLockSQL   = "SELECT GET_LOCK(?, ?)"
	mysqlUnlockSQL = "DO RELEASE_LOCK(?)"
)

type mysqlLocker struct {
	ctx     context.Context
	dbName  string
	appName string

	locker     sync.RWMutex
	lockTimers map[string]struct{}
}

func newMysqlLocker(ctx context.Context, appName, dbName string) Lockable {
	return &mysqlLocker{ctx: ctx, appName: appName, dbName: dbName, lockTimers: map[string]struct{}{}}
}

func (m *mysqlLocker) Lock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	expired := tolerance
	if opt.expired > 0 {
		expired = opt.expired
	}
	lockKey := m.formatLockKey(key)
	if len(lockKey) > 64 {
		return errors.Errorf("key %s length is too long, max key length is 64", lockKey)
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	// disable reentrant
	if _, ok := m.lockTimers[lockKey]; ok {
		return ErrTimeout
	}

	ret := db.Use(ctx, m.dbName, db.AppName(m.appName)).Raw(mysqlLockSQL, lockKey, 0)
	if err = ret.Error; err != nil {
		return ret.Error
	}
	var result int64
	if err = ret.Scan(&result).Error; err != nil {
		return
	}
	if result != 1 {
		return ErrTimeout
	}

	// expire loop
	m.lockTimers[lockKey] = struct{}{}
	timer := time.NewTimer(expired)
	routine.Loop(
		func(ctx context.Context, key string, timer *time.Timer) {
			defer timer.Stop()

			lockKey := m.formatLockKey(key)
			if !m.isLocked(ctx, lockKey) {
				return
			}

			for {
				select {
				case <-ctx.Done():
					_ = m.Unlock(ctx, key) // context done
					return
				case <-m.ctx.Done():
					_ = m.Unlock(ctx, key) // context done
					return
				case <-timer.C:
					_ = m.Unlock(ctx, key) // timeout
					return
				default:
					if !m.isLocked(ctx, lockKey) {
						return
					}
					time.Sleep(200*time.Millisecond + time.Duration(rand.Int63())%(100*time.Millisecond))
				}
			}
		}, routine.Args(ctx, key, timer), routine.AppName(m.appName))
	return
}

func (m *mysqlLocker) Unlock(ctx context.Context, key string, _ ...utils.OptionExtender) (err error) {
	lockKey := m.formatLockKey(key)
	if err = db.Use(ctx, m.dbName, db.AppName(m.appName)).Raw(mysqlUnlockSQL, lockKey).Error; err != nil {
		return
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	delete(m.lockTimers, lockKey)

	return
}

func (m *mysqlLocker) isLocked(ctx context.Context, lockKey string) (locked bool) {
	m.locker.RLock()
	defer m.locker.RUnlock()
	_, locked = m.lockTimers[lockKey]
	return
}

func (m *mysqlLocker) formatLockKey(key string) string {
	return fmt.Sprintf("%s:%s", config.Use(m.appName).AppName(), key)
}
