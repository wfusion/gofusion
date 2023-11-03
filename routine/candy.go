package routine

import (
	"context"
	"log"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

type candyOption struct {
	args    []any
	ch      chan<- any
	wg      *sync.WaitGroup
	appName string
}

func Args(args ...any) utils.OptionFunc[candyOption] {
	return func(o *candyOption) {
		o.args = append(o.args, args...)
	}
}

func WaitGroup(wg *sync.WaitGroup) utils.OptionFunc[candyOption] {
	return func(o *candyOption) {
		o.wg = wg
	}
}

func Channel(ch chan<- any) utils.OptionFunc[candyOption] {
	return func(o *candyOption) {
		o.ch = ch
	}
}

func AppName(name string) utils.OptionFunc[candyOption] {
	return func(o *candyOption) {
		o.appName = name
	}
}

func Go(task any, opts ...utils.OptionExtender) {
	funcName := utils.GetFuncName(task)
	opt := utils.ApplyOptions[candyOption](opts...)
	allocate(opt.appName, 1, &NewPoolOption{ApplyTimeout: -1})
	exec := func() {
		defer func() {
			release(opt.appName, nil, nil)
			delRoutine(opt.appName, funcName)
			if opt.wg != nil {
				opt.wg.Done()
			}
			wg.Done()
		}()

		addRoutine(opt.appName, funcName)
		wrapPromise(task, false, opts...).
			OnFailure(func(v any) {
				if opt.ch == nil {
					log.Printf("[Gofusion] %s catches an error in routine.Go function: \n"+
						"error: %s\nfunc: %s\nfunc signature: %T",
						config.ComponentGoroutinePool, v, utils.GetFuncName(task), task)
				}
			}).
			OnComplete(func(v any) {
				if opt.ch != nil {
					opt.ch <- v
				}
			})
	}

	wg.Add(1)
	if forceSync(opt.appName) {
		exec()
	} else {
		go exec()
	}
}

func Goc(ctx context.Context, task any, opts ...utils.OptionExtender) {
	funcName := utils.GetFuncName(task)
	opt := utils.ApplyOptions[candyOption](opts...)
	allocate(opt.appName, 1, &NewPoolOption{ApplyTimeout: -1})
	exec := func() {
		defer func() {
			release(opt.appName, nil, nil)
			delRoutine(opt.appName, funcName)
			if opt.wg != nil {
				opt.wg.Done()
			}
			wg.Done()
		}()

		addRoutine(opt.appName, funcName)
		select {
		case <-ctx.Done():
		case <-wrapPromise(task, false, opts...).
			OnFailure(func(v any) {
				if opt.ch == nil {
					log.Printf("[Gofusion] %s catches an error in routine.Goc function: \n"+
						"error: %s\nfunc: %s\nfunc signature: %T",
						config.ComponentGoroutinePool, v, utils.GetFuncName(task), task)
				}
			}).
			OnComplete(func(v any) {
				if opt.ch != nil {
					opt.ch <- v
				}
			}).
			GetChan():
		}
	}

	wg.Add(1)
	if forceSync(opt.appName) {
		exec()
	} else {
		go exec()
	}
}

func Loop(task any, opts ...utils.OptionExtender) {
	opt := utils.ApplyOptions[candyOption](opts...)
	allocate(opt.appName, 1, &NewPoolOption{ApplyTimeout: -1}, ignoreRecycled())
	exec := func() {
		defer func() {
			release(opt.appName, nil, ignoreRecycled())
			if opt.wg != nil {
				opt.wg.Done()
			}
		}()
		wrapPromise(task, false, opts...).
			OnFailure(func(v any) {
				if opt.ch == nil {
					log.Printf("[Gofusion] %s catches an error in routine.Loop function: \n"+
						"error: %s\nfunc: %s\nfunc signature: %T",
						config.ComponentGoroutinePool, v, utils.GetFuncName(task), task)
				}
			}).
			OnComplete(func(v any) {
				if opt.ch != nil {
					opt.ch <- v
				}
			})
	}

	go exec()
}

func Loopc(ctx context.Context, task any, opts ...utils.OptionExtender) {
	opt := utils.ApplyOptions[candyOption](opts...)
	allocate(opt.appName, 1, &NewPoolOption{ApplyTimeout: -1}, ignoreRecycled())
	exec := func() {
		defer func() {
			release(opt.appName, nil, ignoreRecycled())
			if opt.wg != nil {
				opt.wg.Done()
			}
		}()
		select {
		case <-ctx.Done():
		case <-wrapPromise(task, false, opts...).
			OnFailure(func(v any) {
				if opt.ch == nil {
					log.Printf("[Gofusion] %s catches an error in routine.Loopc function: \n"+
						"error: %s\nfunc: %s\nfunc signature: %T",
						config.ComponentGoroutinePool, v, utils.GetFuncName(task), task)
				}
			}).
			OnComplete(func(v any) {
				if opt.ch != nil {
					opt.ch <- v
				}
			}).
			GetChan():
		}
	}

	go exec()
}

func Promise(fn any, async bool, opts ...utils.OptionExtender) *Future {
	opt := utils.ApplyOptions[candyOption](opts...)
	funcName := utils.GetFuncName(fn)
	allocate(opt.appName, 1, &NewPoolOption{ApplyTimeout: -1}, ignoreRecycled())
	defer func() {
		release(opt.appName, nil, ignoreRecycled())
		delRoutine(opt.appName, funcName)
		if opt.wg != nil {
			opt.wg.Done()
		}
	}()

	addRoutine(opt.appName, funcName)
	return wrapPromise(fn, async && !forceSync(opt.appName), opts...).
		OnFailure(func(v any) {
			if opt.ch == nil {
				log.Printf("[Gofusion] %s catches an error in routine.Loop function: \n"+
					"error: %s\nfunc: %s\nfunc signature: %T",
					config.ComponentGoroutinePool, v, utils.GetFuncName(fn), fn)
			}
		}).
		OnComplete(func(v any) {
			if opt.ch != nil {
				opt.ch <- v
			}
		})
}

// wrapPromise support function:
// func() error
// func() (any, error)
// func(Canceller)
// func(Canceller) error
// func(Canceller) (any, error)
// func(t1 T1, t2 T2, t3 T3, tx ...Tx)
// func(t1 T1, t2 T2, t3 T3, tx ...Tx) error
// func(t1 T1, t2 T2, t3 T3, tx ...Tx) (any, error)
func wrapPromise(fn any, async bool, opts ...utils.OptionExtender) *Future {
	// check supported function
	switch fn.(type) {
	case func(),
		func() error,
		func() (any, error),
		func(Canceller),
		func(Canceller) error,
		func(Canceller) (any, error):

		return start(fn, async, opts...)

	default:
		typ := reflect.TypeOf(fn)
		if typ.Kind() != reflect.Func {
			return WrapFuture(errors.Errorf("unsupported function type %T", fn), opts...)
		}
		if typ.NumOut() > 0 && (typ.NumOut() > 2 ||
			typ.Out(typ.NumOut()-1) != constant.ErrorType ||
			(typ.NumOut() == 2 && typ.Out(0) != constant.AnyType)) {
			return WrapFuture(errors.Errorf("unsupported function signature %T", fn), opts...)
		}

		return start(fn, async, opts...)
	}
}

// WrapFuture return a Future that presents the wrapped value
func WrapFuture(value any, opts ...utils.OptionExtender) *Future {
	opt := utils.ApplyOptions[candyOption](opts...)
	p := NewPromise()
	p.AppName = opt.appName
	if e, ok := value.(error); !ok {
		_ = p.Resolve(value)
	} else {
		_ = p.Reject(e)
	}

	return p.Future
}

// WhenAny returns a Future.
// If any Future is resolved, this Future will be resolved and return result of resolved Future.
// Otherwise, it will be rejected with results slice returned by all Futures
// Legit types of act are same with Start function
func WhenAny(acts ...any) *Future {
	return WhenAnyMatched(nil, acts...)
}

type anyPromiseResult struct {
	result any
	i      int
}

// WhenAnyMatched returns a Future.
// If any Future is resolved and match the predicate, this Future will be resolved and return result of resolved Future.
// If all Futures are cancelled, this Future will be cancelled.
// Otherwise, it will be rejected with a NoMatchedError included results slice returned by all Futures
// Legit types of act are same with Start function
func WhenAnyMatched(predicate func(any) bool, acts ...any) *Future {
	if predicate == nil {
		predicate = func(v any) bool { return true }
	}

	opts := make([]utils.OptionExtender, 0, len(acts))
	for i, act := range acts {
		if opt, ok := act.(utils.OptionExtender); ok {
			opts = append(opts, opt)
			acts = append(acts[:i], acts[i+1:]...)
		}
	}
	opt := utils.ApplyOptions[candyOption](opts...)
	fs := make([]*Future, len(acts))
	for i, act := range acts {
		fs[i] = Promise(act, true, opts...)
	}

	nf, rs := NewPromise(), make([]any, len(fs))
	if len(acts) == 0 {
		_ = nf.Resolve(nil)
	}

	chFails, chDones := make(chan anyPromiseResult), make(chan anyPromiseResult)

	Go(func() {
		for i, f := range fs {
			k := i
			f.OnSuccess(func(v any) {
				defer func() { _ = recover() }()
				chDones <- anyPromiseResult{result: v, i: k}
			}).OnFailure(func(v any) {
				defer func() { _ = recover() }()
				chFails <- anyPromiseResult{result: v, i: k}
			}).OnCancel(func() {
				defer func() { _ = recover() }()
				chFails <- anyPromiseResult{result: ErrCancelled, i: k}
			})
		}
	}, AppName(opt.appName))

	if len(fs) == 1 {
		select {
		case r := <-chFails:
			if _, ok := r.result.(CancelledError); ok {
				_ = nf.Cancel()
			} else {
				_ = nf.Reject(newNoMatchedError1(r.result))
			}
		case r := <-chDones:
			if predicate(r.result) {
				_ = nf.Resolve(r.result)
			} else {
				_ = nf.Reject(newNoMatchedError1(r.result))
			}
		}
	} else {
		Go(func() {
			defer func() {
				if e := recover(); e != nil {
					_ = nf.Reject(newErrorWithStacks(e))
				}
			}()

			j := 0
			for {
				select {
				case r := <-chFails:
					rs[r.i] = getError(r.result)
				case r := <-chDones:
					if predicate(r.result) {
						// try to cancel other futures
						for _, f := range fs {
							_ = f.Cancel()
						}

						// close the channel for avoid the sender be blocked
						closeChan := func(c chan anyPromiseResult) {
							defer func() { _ = recover() }()
							close(c)
						}
						closeChan(chDones)
						closeChan(chFails)

						// resolve the future and return result
						_ = nf.Resolve(r.result)
						return
					} else {
						rs[r.i] = r.result
					}
				}

				if j++; j == len(fs) {
					m := 0
					for _, r := range rs {
						switch val := r.(type) {
						case CancelledError:
						default:
							m++
							_ = val
						}
					}
					if m > 0 {
						_ = nf.Reject(newNoMatchedError(rs))
					} else {
						_ = nf.Cancel()
					}
					break
				}
			}
		}, AppName(opt.appName))
	}
	return nf.Future
}

// WhenAll receives function slice and returns a Future.
// If all Futures are resolved, this Future will be resolved and return results slice.
// Otherwise, it will be rejected with results slice returned by all Futures
// Legit types of act are same with Start function
func WhenAll(acts ...any) (fu *Future) {
	p := NewPromise()
	fu = p.Future

	opts := make([]utils.OptionExtender, 0, len(acts))
	for i, act := range acts {
		if opt, ok := act.(utils.OptionExtender); ok {
			opts = append(opts, opt)
			acts = append(acts[:i], acts[i+1:]...)
		}
	}
	opt := utils.ApplyOptions[candyOption](opts...)
	p.AppName = opt.appName
	if len(acts) == 0 {
		_ = p.Resolve([]any{})
		return
	}

	fs := make([]*Future, len(acts))
	for i, act := range acts {
		fs[i] = Promise(act, true, opts...)
	}
	fu = whenAllFuture(fs, opts...)
	return
}

// WhenAll receives Futures slice and returns a Future.
// If all Futures are resolved, this Future will be resolved and return results slice.
// If any Future is cancelled, this Future will be cancelled.
// Otherwise, it will be rejected with results slice returned by all Futures.
// Legit types of act are same with Start function
func whenAllFuture(fs []*Future, opts ...utils.OptionExtender) *Future {
	opt := utils.ApplyOptions[candyOption](opts...)
	wf := NewPromise()
	wf.AppName = opt.appName
	rs := make([]any, len(fs))

	if len(fs) == 0 {
		_ = wf.Resolve([]any{})
	} else {
		n := int32(len(fs))
		cancelOthers := func(j int) {
			for k, f1 := range fs {
				if k != j {
					_ = f1.Cancel()
				}
			}
		}

		Go(func() {
			isCancelled := int32(0)
			for i, f := range fs {
				j := i

				f.OnSuccess(func(v any) {
					rs[j] = v
					if atomic.AddInt32(&n, -1) == 0 {
						_ = wf.Resolve(rs)
					}
				}).OnFailure(func(v any) {
					if atomic.CompareAndSwapInt32(&isCancelled, 0, 1) {
						// try to cancel all futures
						cancelOthers(j)

						// errs := make([]error, 0, 1)
						// errs = append(errs, v.(error))
						e := newAggregateError1("error appears in WhenAll:", v)
						_ = wf.Reject(e)
					}
				}).OnCancel(func() {
					if atomic.CompareAndSwapInt32(&isCancelled, 0, 1) {
						// try to cancel all futures
						cancelOthers(j)

						_ = wf.Cancel()
					}
				})
			}
		}, AppName(opt.appName))
	}

	return wf.Future
}

// start a goroutines to execute task function
// and return a Future that presents the result.
// If option parameter is true, the act function will be sync called.
// Type of act can be any of below four types:
//
//	func() (r any, err error):
//	   if err returned by act != nil or panic error, then Future will be rejected with error,
//	   otherwise be resolved with r.
//	func():
//	   if act panic error, then Future will be rejected, otherwise be resolved with nil.
//	func(c promise.Canceller) (r any, err error):
//	   if err returned by act != nil or panic error,
//	   then Future will be rejected with err, otherwise be resolved with r.
//	   We can check c.IsCancelled() to decide whether we need to exit act function
//	func(promise.Canceller):
//	   if act panic error, then Future will be rejected with error, otherwise be resolved with nil.
//	   We can check c.IsCancelled() to decide whether we need to exit act function.
//	error:
//	   Future will be rejected with error immediately
//	other value:
//	   Future will be resolved with value immediately
func start(act any, async bool, opts ...utils.OptionExtender) *Future {
	p := NewPromise()
	if f, ok := act.(*Future); ok {
		return f
	}

	opt := utils.ApplyOptions[candyOption](opts...)
	p.AppName = opt.appName
	if action := getAct(p, act); action != nil {
		if !async {
			// sync call
			r, err := action(opt)
			if p.IsCancelled() {
				_ = p.Cancel()
			} else {
				if err == nil {
					_ = p.Resolve(r)
				} else {
					_ = p.Reject(err)
				}
			}
		} else {
			// async call
			Go(func() {
				r, err := action(opt)
				if p.IsCancelled() {
					_ = p.Cancel()
				} else {
					if err == nil {
						_ = p.Resolve(r)
					} else {
						_ = p.Reject(err)
					}
				}
			}, AppName(opt.appName))
		}
	}

	return p.Future
}
