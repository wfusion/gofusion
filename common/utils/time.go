package utils

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/multierr"
)

// UnixNano returns t as a Unix time, the number of nanoseconds elapsed
// since January 1, 1970 UTC. The result is undefined if the Unix time
// in nanoseconds cannot be represented by an int64 (a date before the year
// 1678 or after 2262). Note that this means the result of calling UnixNano
// on the zero Time is undefined. The result does not depend on the
// location associated with t.
const (
	minYear = 1678
	maxYear = 2262
)

// GetTime 将毫秒级的时间戳转换成时间
func GetTime(timestampMs int64) time.Time {
	return time.UnixMilli(timestampMs)
}

// GetTimeStamp 将时间转换成毫秒级的时间戳
func GetTimeStamp(t time.Time) int64 {
	if year := t.Year(); year >= maxYear || year < minYear {
		return t.Unix() * 1e3
	}
	return t.UnixNano() / 1e6
}

// IsValidTimestamp 返回 false 表示无法对毫秒时间戳和 time.Time 进行精确转换
func IsValidTimestamp(timeMS int64) bool {
	year := GetTime(timeMS).Year()
	return year >= minYear && year < maxYear
}

type loopWithIntervalOption struct {
	maxTimes uint
	// jitter time
	base, max  time.Duration
	ratio, exp float64
	symmetric  bool
}

// LoopJitterInterval Deprecated, try github.com/rican7/retry.Retry instead
func LoopJitterInterval(base, max time.Duration, ratio, exp float64,
	symmetric bool) OptionFunc[loopWithIntervalOption] {
	return func(o *loopWithIntervalOption) {
		o.base = base
		o.max = max
		o.ratio = ratio
		o.exp = exp
		o.symmetric = symmetric
	}
}

// LoopMaxTimes Deprecated, try github.com/rican7/retry.Retry instead
func LoopMaxTimes(maxTimes uint) OptionFunc[loopWithIntervalOption] {
	return func(o *loopWithIntervalOption) {
		o.maxTimes = maxTimes
	}
}

// LoopWithInterval Deprecated, try github.com/rican7/retry.Retry instead
func LoopWithInterval(ctx context.Context, interval time.Duration,
	fn func() bool, opts ...OptionExtender) (err error) {
	var (
		maxTimes     uint
		nextInterval func() time.Duration
	)

	opt := ApplyOptions[loopWithIntervalOption](opts...)
	enableJitter := opt.base > 0
	enableMaxTimes := opt.maxTimes > 0
	if enableJitter {
		nextInterval = NextJitterIntervalFunc(opt.base, opt.max, opt.ratio, opt.exp, opt.symmetric)
		interval = nextInterval()
	}
	if enableMaxTimes {
		maxTimes = opt.maxTimes
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		if fn() {
			return
		}

		if enableMaxTimes {
			if maxTimes--; maxTimes == 0 {
				return multierr.Append(err, errors.New("exceed the maximum times"))
			}
		}

		// time.Sleep
		timer.Reset(interval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if enableJitter {
				interval = nextInterval()
			}
		}
	}
}

// NextJitterIntervalFunc generate a jitter and exponential power duration, inspired by net/http.(*Server).Shutdown
func NextJitterIntervalFunc(base, max time.Duration, ratio, exp float64, symmetric bool) func() time.Duration {
	return func() (interval time.Duration) {
		// add specified ratio jitter
		if !symmetric {
			// if ratio is 0.5
			// then interval = base + random(0.5*base) <=> [base, 1.5*base)
			// if ratio is 0.1
			// then interval = base + random(0.1*base) <=> [base, 1.1*base)
			_range := float64(base) * ratio * rand.Float64()
			interval = base + time.Duration(_range)
		} else {
			// if ratio is 0.5
			// then interval = base + random(0.5*base) - 0.25*base <=> [0.75*base, 1.25*base)
			// if ratio is 0.1
			// then interval = base + random(0.1*base) - 0.05*base <=> [0.95*base, 1.05*base)
			_range := float64(base) * ratio * rand.Float64()
			interval = base + time.Duration(_range) - time.Duration(float64(base)*(ratio/2))
		}

		// double and clamp for next time
		base = time.Duration(float64(base) * exp)
		if base > max {
			base = max
		}
		return interval
	}
}

// WaitGroupTimeout adds timeout feature for sync.WaitGroup.Wait().
// It returns true, when timeout.
type timeoutOption struct {
	wg *sync.WaitGroup
}

func TimeoutWg(wg *sync.WaitGroup) OptionFunc[timeoutOption] {
	return func(o *timeoutOption) {
		o.wg = wg
	}
}

func Timeout(timeout time.Duration, opts ...OptionExtender) bool {
	opt := ApplyOptions[timeoutOption](opts...)
	wgClosed := make(chan struct{}, 1)
	go func() {
		switch {
		case opt.wg != nil:
			opt.wg.Wait()
		}
		wgClosed <- struct{}{}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-wgClosed:
		return false
	case <-timer.C:
		return true
	}
}
