package routine

import (
	"math/rand"
	"unsafe"
)

var (
	ErrCancelled error = &CancelledError{}
)

// CancelledError present the Future object is cancelled.
type CancelledError struct {
}

func (e *CancelledError) Error() string {
	return "Task be cancelled"
}

// resultType present the type of Future final status.
type resultType int

const (
	ResultSuccess resultType = iota
	ResultFailure
	ResultCancelled
)

// Result presents the result of a promise.
// If Typ is ResultSuccess, Result field will present the returned value of Future task.
// If Typ is ResultFailure, Result field will present a related error .
// If Typ is ResultCancelled, Result field will be null.
type Result struct {
	Result any        //result of the promise
	Typ    resultType //success, failure, or cancelled?
}

// promise presents an object that acts as a proxy for a result.
// that is initially unknown, usually because the computation of its
// value is yet incomplete (refer to wikipedia).
// You can use Resolve/Reject/Cancel to set the final result of promise.
// Future can return a read-only placeholder view of result.
type promise struct {
	*Future
}

// Cancel sets the status of promise to ResultCancelled.
// If promise is cancelled, Get() will return nil and CANCELLED error.
// All callback functions will be not called if promise is cancelled.
func (p *promise) Cancel() (e error) {
	return p.Future.Cancel()
}

// Resolve sets the value for promise, and the status will be changed to ResultSuccess.
// if promise is resolved, Get() will return the value and nil error.
func (p *promise) Resolve(v any) (e error) {
	return p.setResult(&Result{v, ResultSuccess})
}

// Reject sets the error for promise, and the status will be changed to ResultFailure.
// if promise is rejected, Get() will return nil and the related error value.
func (p *promise) Reject(err error) (e error) {
	return p.setResult(&Result{err, ResultFailure})
}

// OnSuccess registers a callback function that will be called when promise is resolved.
// If promise is already resolved, the callback will immediately be called.
// The value of promise will be parameter of Done callback function.
func (p *promise) OnSuccess(callback func(v any)) *promise {
	p.Future.OnSuccess(callback)
	return p
}

// OnFailure registers a callback function that will be called when promise is rejected.
// If promise is already rejected, the callback will immediately be called.
// The error of promise will be parameter of Fail callback function.
func (p *promise) OnFailure(callback func(v any)) *promise {
	p.Future.OnFailure(callback)
	return p
}

// OnComplete register a callback function that will be called when promise is rejected or resolved.
// If promise is already rejected or resolved, the callback will immediately be called.
// According to the status of promise, value or error will be parameter of Always callback function.
// Value is the parameter if promise is resolved, or error is the parameter if promise is rejected.
// Always callback will be not called if promise be called.
func (p *promise) OnComplete(callback func(v any)) *promise {
	p.Future.OnComplete(callback)
	return p
}

// OnCancel registers a callback function that will be called when promise is cancelled.
// If promise is already cancelled, the callback will immediately be called.
func (p *promise) OnCancel(callback func()) *promise {
	p.Future.OnCancel(callback)
	return p
}

// NewPromise is factory function for promise
func NewPromise() *promise {
	val := &futureVal{
		dones:   make([]func(v any), 0, 8),
		fails:   make([]func(v any), 0, 8),
		always:  make([]func(v any), 0, 4),
		cancels: make([]func(), 0, 2),
		pipes:   make([]*pipe, 0, 4),
	}
	f := &promise{
		Future: &Future{
			Id:    rand.Int(),
			final: make(chan struct{}),
			val:   unsafe.Pointer(val),
		},
	}
	return f
}
