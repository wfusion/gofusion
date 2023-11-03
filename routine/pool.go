package routine

import (
	"math"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/utils"
)

var (
	// pools is a global map of goroutine pools, managing multiple goroutine pools,
	// with total service instance quantity controlled by allocated
	// TODO: Issue with nested goroutine pool allocation;
	//       if a task executed in a goroutine pool nests a new goroutine pool allocation,
	//       it might cause a deadlock, and errors cannot be quickly thrown.
	pools         map[string]map[string]Pool
	rwlock        sync.RWMutex
	ignored       map[string]*atomic.Int64
	allocated     map[string]*atomic.Int64
	defaultLogger map[string]ants.Logger
)

type pool struct {
	appName string
	name    string
	pool    *ants.Pool
	option  *NewPoolOption
}

func (p *pool) Submit(task any, opts ...utils.OptionExtender) (e error) {
	opt := utils.ApplyOptions[candyOption](opts...)
	wrapFn := utils.WrapFunc1[error](task)
	if !forceSync(p.appName) {
		return p.pool.Submit(func() { e = wrapFn(opt.args...) })
	}

	return wrapFn(opt.args...)
}
func (p *pool) Running() int   { return p.pool.Running() }
func (p *pool) Free() int      { return p.pool.Free() }
func (p *pool) Waiting() int   { return p.pool.Waiting() }
func (p *pool) Cap() int       { return p.pool.Cap() }
func (p *pool) IsClosed() bool { return p.pool.IsClosed() }
func (p *pool) Release(opts ...utils.OptionExtender) {
	defer release(p.appName, p, opts...)
	p.pool.Release()
}
func (p *pool) ReleaseTimeout(timeout time.Duration, opts ...utils.OptionExtender) error {
	defer release(p.appName, p, opts...)
	return p.pool.ReleaseTimeout(timeout)
}

func NewPool(name string, size int, opts ...utils.OptionExtender) (p Pool) {
	o := utils.ApplyOptions[NewPoolOption](opts...)
	opt := utils.ApplyOptions[candyOption](opts...)
	if o.Logger == nil {
		o.Logger = defaultLogger[opt.appName]
	}

	validate(opt.appName, name)
	allocate(opt.appName, size, o)

	antsPool, err := ants.NewPool(size, ants.WithOptions(ants.Options{
		ExpiryDuration:   o.ExpiryDuration,
		PreAlloc:         o.PreAlloc,
		MaxBlockingTasks: o.MaxBlockingTasks,
		Nonblocking:      o.Nonblocking,
		PanicHandler:     o.PanicHandler,
		Logger:           o.Logger,
		DisablePurge:     o.DisablePurge,
	}))
	if err != nil {
		panic(err)
	}

	p = &pool{appName: opt.appName, name: name, pool: antsPool, option: o}
	addPool(opt.appName, name, p)
	return
}

type internalOption struct {
	// ignoreRecycled does not account for unrecycled successes during graceful exit,
	// only used when calling the loop method
	ignoreRecycled bool
	// ignoreMutex does not lock on map operations,
	// considering replacing with reentrant lock github.com/sasha-s/go-deadlock,
	// but introduction will increase comprehension cost, also goes against go design
	ignoreMutex bool
}

func ignoreRecycled() utils.OptionFunc[internalOption] {
	return func(o *internalOption) {
		o.ignoreRecycled = true
	}
}

func ignoreMutex() utils.OptionFunc[internalOption] {
	return func(o *internalOption) {
		o.ignoreMutex = true
	}
}

func allocate(appName string, size int, o *NewPoolOption, opts ...utils.OptionExtender) {
	oo := utils.ApplyOptions[internalOption](opts...)
	demands := int64(size)
	rwlock.RLock()
	defer rwlock.RUnlock()
	if allocated[appName].Load()-demands < 0 && o.ApplyTimeout == 0 {
		panic(ErrPoolOverload)
	}

	if o.ApplyTimeout < 0 {
		o.ApplyTimeout = math.MaxInt64
	}

	t := time.NewTimer(o.ApplyTimeout)
	for {
		select {
		case <-t.C:
			panic(ErrTimeout)
		default:
			minuend := allocated[appName].Load()
			diff := minuend - demands
			// main thread is a goroutine as well, so diff should be greater than 0
			if diff <= 0 || !allocated[appName].CompareAndSwap(minuend, diff) {
				continue
			}
			if oo.ignoreRecycled {
				ignored[appName].Add(demands)
			}

			return
		}
	}
}

func release(appName string, p *pool, opts ...utils.OptionExtender) {
	o := utils.ApplyOptions[internalOption](opts...)
	capacity := int64(1)

	if p != nil {
		if !o.ignoreMutex {
			rwlock.Lock()
			defer rwlock.Unlock()
		}
		delete(pools[appName], p.name)
		capacity = int64(p.pool.Cap())
	}

	alloc := allocated[appName]
	if alloc == nil {
		return
	}
	alloc.Add(capacity)
	if o != nil && o.ignoreRecycled {
		alloc.Sub(capacity)
	}
}

func addPool(appName, name string, pool Pool) {
	rwlock.Lock()
	defer rwlock.Unlock()
	pools[appName][name] = pool
}

func validate(appName, name string) {
	rwlock.RLock()
	defer rwlock.RUnlock()
	if _, ok := pools[appName][name]; ok {
		panic(ErrDuplicatedName)
	}
}

type PoolOption struct {
	// ExpiryDuration is a period for the scavenger goroutine to clean up those expired workers,
	// the scavenger scans all workers every `ExpiryDuration` and clean up those workers that haven't been
	// used for more than `ExpiryDuration`.
	ExpiryDuration time.Duration

	// PreAlloc indicates whether to make memory pre-allocation when initializing Pool.
	PreAlloc bool

	// Max number of goroutine blocking on pool.Submit.
	// 0 (default value) means no such limit.
	MaxBlockingTasks int

	// When Nonblocking is true, Pool.Submit will never be blocked.
	// ErrPoolOverload will be returned when Pool.Submit cannot be done at once.
	// When Nonblocking is true, MaxBlockingTasks is inoperative.
	Nonblocking bool

	// PanicHandler is used to handle panics from each worker goroutine.
	// if nil, panics will be thrown out again from worker goroutines.
	PanicHandler func(any)

	// Logger is the customized logger for logging info, if it is not set,
	// default standard logger from log package is used.
	Logger ants.Logger

	// When DisablePurge is true, workers are not purged and are resident.
	DisablePurge bool
}

type NewPoolOption struct {
	PoolOption
	// ApplyTimeout is the timeout duration for applying a goroutine pool
	// Default = 0 means non-blocking and directly panic;
	// < 0 means blocking and wait;
	// > 0 means block and panic after timeout
	ApplyTimeout time.Duration
}

func Timeout(t time.Duration) utils.OptionFunc[NewPoolOption] {
	return func(o *NewPoolOption) {
		o.ApplyTimeout = t
	}
}

func WithoutTimeout() utils.OptionFunc[NewPoolOption] {
	return func(o *NewPoolOption) {
		o.ApplyTimeout = -1
	}
}

func Options(in *NewPoolOption) utils.OptionFunc[NewPoolOption] {
	return func(o *NewPoolOption) {
		o.ApplyTimeout = in.ApplyTimeout
		o.ExpiryDuration = in.PoolOption.ExpiryDuration
		o.PreAlloc = in.PreAlloc
		o.MaxBlockingTasks = in.MaxBlockingTasks
		o.Nonblocking = in.Nonblocking
		o.PanicHandler = in.PanicHandler
		o.Logger = in.Logger
		o.DisablePurge = in.DisablePurge
	}
}
