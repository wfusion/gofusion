package lock

import (
	"context"
	"time"

	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/routine"
)

func Within(ctx context.Context, locker Lockable, key string,
	expired, timeout time.Duration, cb func() error, opts ...utils.OptionExtender) (err error) {
	const (
		reLockWaitTime = time.Duration(200) * time.Millisecond
	)
	opt := utils.ApplyOptions[useOption](opts...)
	optL := utils.ApplyOptions[lockOption](opts...)
	if optL.reentrantKey == "" {
		optL.reentrantKey = utils.ULID()
	}
	optionals := []utils.OptionExtender{ReentrantKey(optL.reentrantKey)}
	if expired > 0 {
		optionals = append(optionals, Expire(expired))
	}

	done := make(chan struct{}, 1)
	timeFault := make(chan struct{}, 1)
	routine.Goc(ctx, func() {
		defer func() { done <- struct{}{} }()

		var e error
		rLocker, ok := locker.(ReentrantLockable)
		for {
			select {
			case <-timeFault: // timeout exit
				return
			default:
				if ok {
					if e = rLocker.ReentrantLock(ctx, key, optL.reentrantKey, optionals...); e == nil {
						return
					}
				} else {
					if e = locker.Lock(ctx, key, optionals...); e == nil {
						return
					}
				}

				// relock after 200 milliseconds
				time.Sleep(reLockWaitTime)
			}
		}
	}, routine.AppName(opt.appName))

	timer := time.NewTimer(timeout)
	select {
	// success
	case <-done:

	// context done
	case <-ctx.Done():
		timeFault <- struct{}{}
		return ErrContextDone

	// timeout
	case <-timer.C:
		timeFault <- struct{}{}
		return ErrTimeout
	}

	defer func() { err = multierr.Append(err, locker.Unlock(ctx, key, optionals...)) }()

	_, err = utils.Catch(cb)
	return
}
