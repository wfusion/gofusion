package routine

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"
	"unsafe"
)

type callbackType int

const (
	CallbackDone callbackType = 1 + iota
	CallbackFail
	CallbackAlways
	CallbackCancel
)

// pipe presents a promise that will be chain call
type pipe struct {
	pipeDoneTask, pipeFailTask func(v any) *Future
	pipePromise                *promise
}

// getPipe returns piped Future task function and pipe promise by the status of current promise.
func (p *pipe) getPipe(isResolved bool) (func(v any) *Future, *promise) {
	if isResolved {
		return p.pipeDoneTask, p.pipePromise
	} else {
		return p.pipeFailTask, p.pipePromise
	}
}

// Canceller is used to check if the future is cancelled
// It be usually passed to the future task function
// for future task function can check if the future is cancelled.
type Canceller interface {
	IsCancelled() bool
	Cancel()
}

// canceller provides an implement of Canceller interface.
// It will be passed to future task function as parameter
type canceller struct {
	f *Future
}

// Cancel sets Future task to CANCELLED status
func (c *canceller) Cancel() {
	_ = c.f.Cancel()
}

// IsCancelled returns true if Future task is cancelled, otherwise false.
func (c *canceller) IsCancelled() (r bool) {
	return c.f.IsCancelled()
}

// futureVal stores the internal state of Future.
type futureVal struct {
	dones, fails, always []func(v any)
	cancels              []func()
	pipes                []*pipe
	r                    *Result
}

// Future provides a read-only view of promise,
// the value is set by using Resolve, Reject and Cancel methods of related promise
type Future struct {
	Id      int // ID can be used as identity of Future
	AppName string

	final chan struct{}
	// val point to futureVal that stores status of future
	// if we need to change the status of future, must copy a new futureVal and modify it,
	// then use CAS to put the pointer of new futureVal
	val unsafe.Pointer
}

// Canceller returns a canceller object related to future.
func (f *Future) Canceller() Canceller {
	return &canceller{f}
}

// IsCancelled returns true if the promise is cancelled, otherwise false
func (f *Future) IsCancelled() bool {
	val := f.loadVal()

	if val != nil && val.r != nil && val.r.Typ == ResultCancelled {
		return true
	} else {
		return false
	}
}

// SetTimeout sets the future task will be cancelled
// if future is not complete before time out
func (f *Future) SetTimeout(mm int) *Future {
	if mm == 0 {
		mm = 10
	} else {
		mm = mm * 1000 * 1000
	}

	Go(func() {
		<-time.After((time.Duration)(mm) * time.Nanosecond)
		_ = f.Cancel()
	}, AppName(f.AppName))
	return f
}

// GetChan returns a channel than can be used to receive result of promise
func (f *Future) GetChan() <-chan *Result {
	c := make(chan *Result, 1)
	f.OnComplete(func(v any) {
		c <- f.loadResult()
	}).OnCancel(func() {
		c <- f.loadResult()
	})
	return c
}

// Get will block current goroutines until the Future is resolved/rejected/cancelled.
// If Future is resolved, value and nil will be returned
// If Future is rejected, nil and error will be returned.
// If Future is cancelled, nil and CANCELLED error will be returned.
func (f *Future) Get() (val any, err error) {
	<-f.final
	return getFutureReturnVal(f.loadResult())
}

// GetOrTimeout is similar to Get(), but GetOrTimeout will not block after timeout.
// If GetOrTimeout returns with a timeout, timeout value will be true in return values.
// The unit of parameter is millisecond.
func (f *Future) GetOrTimeout(mm uint) (val any, err error, timout bool) {
	if mm == 0 {
		mm = 10
	} else {
		mm = mm * 1000 * 1000
	}

	select {
	case <-time.After((time.Duration)(mm) * time.Nanosecond):
		return nil, nil, true
	case <-f.final:
		r, err := getFutureReturnVal(f.loadResult())
		return r, err, false
	}
}

// Cancel sets the status of promise to ResultCancelled.
// If promise is cancelled, Get() will return nil and CANCELLED error.
// All callback functions will be not called if promise is cancelled.
func (f *Future) Cancel() (e error) {
	return f.setResult(&Result{ErrCancelled, ResultCancelled})
}

// OnSuccess registers a callback function that will be called when promise is resolved.
// If promise is already resolved, the callback will immediately be called.
// The value of promise will be parameter of Done callback function.
func (f *Future) OnSuccess(callback func(v any)) *Future {
	f.addCallback(callback, CallbackDone)
	return f
}

// OnFailure registers a callback function that will be called when promise is rejected.
// If promise is already rejected, the callback will immediately be called.
// The error of promise will be parameter of Fail callback function.
func (f *Future) OnFailure(callback func(v any)) *Future {
	f.addCallback(callback, CallbackFail)
	return f
}

// OnComplete register a callback function that will be called when promise is rejected or resolved.
// If promise is already rejected or resolved, the callback will immediately be called.
// According to the status of promise, value or error will be parameter of Always callback function.
// Value is the parameter if promise is resolved, or error is the parameter if promise is rejected.
// Always callback will be not called if promise be called.
func (f *Future) OnComplete(callback func(v any)) *Future {
	f.addCallback(callback, CallbackAlways)
	return f
}

// OnCancel registers a callback function that will be called when promise is cancelled.
// If promise is already cancelled, the callback will immediately be called.
func (f *Future) OnCancel(callback func()) *Future {
	f.addCallback(callback, CallbackCancel)
	return f
}

// Pipe registers one or two functions that returns a Future, and returns a proxy of pipeline Future.
// First function will be called when Future is resolved, the returned Future will be as pipeline Future.
// Secondary function will be called when Future is rejected, the returned Future will be as pipeline Future.
func (f *Future) Pipe(callbacks ...any) (result *Future, ok bool) {
	if len(callbacks) == 0 ||
		(len(callbacks) == 1 && callbacks[0] == nil) ||
		(len(callbacks) > 1 && callbacks[0] == nil && callbacks[1] == nil) {
		result = f
		return
	}

	// ensure all callback functions match the spec "func(v any) *Future"
	cs := make([]func(v any) *Future, len(callbacks), len(callbacks))
	for i, callback := range callbacks {
		switch c := callback.(type) {
		case func(v any) *Future:
			cs[i] = c
		case func() *Future:
			cs[i] = func(v any) *Future {
				return c()
			}
		case func(v any):
			cs[i] = func(v any) *Future {
				return start(func() {
					c(v)
				}, true)
			}
		case func(v any) (r any, err error):
			cs[i] = func(v any) *Future {
				return start(func() (r any, err error) {
					r, err = c(v)
					return
				}, true)
			}
		case func():
			cs[i] = func(v any) *Future {
				return start(func() {
					c()
				}, true)
			}
		case func() (r any, err error):
			cs[i] = func(v any) *Future {
				return start(func() (r any, err error) {
					r, err = c()
					return
				}, true)
			}
		default:
			ok = false
			return
		}
	}

	for {
		v := f.loadVal()
		r := v.r
		if r != nil {
			result = f
			if r.Typ == ResultSuccess && cs[0] != nil {
				result = cs[0](r.Result)
			} else if r.Typ == ResultFailure && len(cs) > 1 && cs[1] != nil {
				result = cs[1](r.Result)
			}
		} else {
			newPipe := &pipe{}
			newPipe.pipeDoneTask = cs[0]
			if len(cs) > 1 {
				newPipe.pipeFailTask = cs[1]
			}
			newPipe.pipePromise = NewPromise()

			newVal := *v
			newVal.pipes = append(newVal.pipes, newPipe)

			//use CAS to ensure that the state of Future is not changed,
			//if the state is changed, will retry CAS operation.
			if atomic.CompareAndSwapPointer(&f.val, unsafe.Pointer(v), unsafe.Pointer(&newVal)) {
				result = newPipe.pipePromise.Future
				break
			}
		}
	}

	ok = true
	return
}

// result uses Atomic load to return result of the Future
func (f *Future) loadResult() *Result {
	val := f.loadVal()
	return val.r
}

// val uses Atomic load to return state value of the Future
func (f *Future) loadVal() *futureVal {
	r := atomic.LoadPointer(&f.val)
	return (*futureVal)(r)
}

// setResult sets the value and final status of promise, it will only be executed for once
func (f *Future) setResult(r *Result) (e error) {
	defer func() {
		if err := getError(recover()); err != nil {
			e = err
			fmt.Printf("\nerror in setResult(): %s\n%s\n", err, debug.Stack())
		}
	}()

	e = errors.New("cannot resolve/reject/cancel more than once")

	for {
		v := f.loadVal()
		if v.r != nil {
			return
		}
		newVal := *v
		newVal.r = r

		// Use CAS operation to ensure that the state of promise isn't changed.
		// If the state is changed, must get the latest state and try to call CAS again.
		// No ABA issue in this case because address of all objects are different.
		if atomic.CompareAndSwapPointer(&f.val, unsafe.Pointer(v), unsafe.Pointer(&newVal)) {
			//Close chEnd then all Get() and GetOrTimeout() will be unblocked
			close(f.final)

			//call callback functions and start the promise pipeline
			if len(v.dones) > 0 || len(v.fails) > 0 || len(v.always) > 0 || len(v.cancels) > 0 {
				Go(func() {
					execCallback(r, v.dones, v.fails, v.always, v.cancels)
				}, AppName(f.AppName))
			}

			//start the pipeline
			if len(v.pipes) > 0 {
				Go(func() {
					for _, pipe := range v.pipes {
						pipeTask, pipePromise := pipe.getPipe(r.Typ == ResultSuccess)
						startPipe(r, pipeTask, pipePromise)
					}
				}, AppName(f.AppName))
			}
			e = nil
			break
		}
	}
	return
}

// handleOneCallback registers a callback function
func (f *Future) addCallback(callback any, t callbackType) {
	if callback == nil {
		return
	}
	if (t == CallbackDone) ||
		(t == CallbackFail) ||
		(t == CallbackAlways) {
		if _, ok := callback.(func(v any)); !ok {
			panic(errors.New("callback function spec must be func(v any)"))
		}
	} else if t == CallbackCancel {
		if _, ok := callback.(func()); !ok {
			panic(errors.New("callback function spec must be func()"))
		}
	}

	for {
		v := f.loadVal()
		r := v.r
		if r == nil {
			newVal := *v
			switch t {
			case CallbackDone:
				newVal.dones = append(newVal.dones, callback.(func(v any)))
			case CallbackFail:
				newVal.fails = append(newVal.fails, callback.(func(v any)))
			case CallbackAlways:
				newVal.always = append(newVal.always, callback.(func(v any)))
			case CallbackCancel:
				newVal.cancels = append(newVal.cancels, callback.(func()))
			}

			//use CAS to ensure that the state of Future is not changed,
			//if the state is changed, will retry CAS operation.
			if atomic.CompareAndSwapPointer(&f.val, unsafe.Pointer(v), unsafe.Pointer(&newVal)) {
				break
			}
		} else {
			if (t == CallbackDone && r.Typ == ResultSuccess) ||
				(t == CallbackFail && r.Typ == ResultFailure) ||
				(t == CallbackAlways && r.Typ != ResultCancelled) {
				callbackFunc := callback.(func(v any))
				callbackFunc(r.Result)
			} else if t == CallbackCancel && r.Typ == ResultCancelled {
				callbackFunc := callback.(func())
				callbackFunc()
			}
			break
		}
	}
}
