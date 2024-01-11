package routine

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"

	"github.com/wfusion/gofusion/common/utils"
)

// NoMatchedError presents no future that returns matched result in WhenAnyTrue function.
type NoMatchedError struct {
	Results []any
}

func (e *NoMatchedError) Error() string {
	return "No matched future"
}

func (e *NoMatchedError) HasError() bool {
	for _, ie := range e.Results {
		if _, ok1 := ie.(error); ok1 {
			return true
		}
	}
	return false
}

func newNoMatchedError(results []any) *NoMatchedError {
	return &NoMatchedError{results}
}

func newNoMatchedError1(e any) *NoMatchedError {
	return &NoMatchedError{[]any{e}}
}

// AggregateError aggregate multi errors into an error
type AggregateError struct {
	s         string
	InnerErrs []error
}

func (e *AggregateError) Error() string {
	if e.InnerErrs == nil {
		return e.s
	} else {
		buf := bytes.NewBufferString(e.s)
		_, _ = buf.WriteString("\n\n")
		for i, ie := range e.InnerErrs {
			if ie == nil {
				continue
			}
			_, _ = buf.WriteString("error appears in Future ")
			_, _ = buf.WriteString(strconv.Itoa(i))
			_, _ = buf.WriteString(": ")
			_, _ = buf.WriteString(ie.Error())
			_, _ = buf.WriteString("\n")
		}
		_, _ = buf.WriteString("\n")
		return buf.String()
	}
}

func newAggregateError1(s string, e any) *AggregateError {
	return &AggregateError{newErrorWithStacks(s).Error(), []error{getError(e)}}
}

func newErrorWithStacks(i any) (e error) {
	err := getError(i)
	buf := bytes.NewBufferString(err.Error())
	_, _ = buf.WriteString("\n")

	pcs := make([]uintptr, 50)
	num := runtime.Callers(2, pcs)
	for _, v := range pcs[0:num] {
		fun := runtime.FuncForPC(v)
		file, line := fun.FileLine(v)
		name := fun.Name()
		writeStrings(buf, []string{name, " ", file, ":", strconv.Itoa(line), "\n"})
	}
	return errors.New(buf.String())
}

func getAct(p *promise, act any) (f func(opt *candyOption) (r any, err error)) {
	var (
		act1 func(...any) (any, error)
		act2 func(Canceller, ...any) (any, error)
	)
	canCancel := false

	// convert the act to the function that has return value and error if act function haven't return value and error
	switch v := act.(type) {
	case func():
		act1 = func(...any) (any, error) { v(); return nil, nil }
	case func() error:
		act1 = func(...any) (any, error) { e := v(); return nil, e }
	case func() (any, error):
		act1 = func(a ...any) (any, error) { return v() }
	case func(Canceller):
		canCancel = true
		act2 = func(canceller Canceller, a ...any) (any, error) { v(canceller); return nil, nil }
	case func(Canceller) error:
		canCancel = true
		act2 = func(canceller Canceller, a ...any) (any, error) { e := v(canceller); return nil, e }
	case func(Canceller) (any, error):
		canCancel = true
		act2 = func(canceller Canceller, a ...any) (any, error) { return v(canceller) }
	default:
		typ := reflect.TypeOf(v)
		if typ.Kind() == reflect.Func {
			act1 = utils.WrapFunc2[any, error](v)
		} else {
			if e, ok := v.(error); !ok {
				_ = p.Resolve(v)
			} else {
				_ = p.Reject(e)
			}
			return nil
		}
	}

	// If parameters of act function has a Canceller interface, the Future will be cancelled.
	var canceller Canceller
	if p != nil && canCancel {
		canceller = p.Canceller()
	}

	// return proxy function of act function
	f = func(opt *candyOption) (r any, err error) {
		defer func() {
			if e := recover(); e != nil {
				err = newErrorWithStacks(e)
			}
		}()

		if canCancel {
			r, err = act2(canceller, opt.args...)
		} else {
			r, err = act1(opt.args...)
		}

		return
	}
	return
}

// Error handling struct and functions------------------------------
type stringer interface {
	String() string
}

func getError(i any) (e error) {
	if i != nil {
		switch v := i.(type) {
		case error:
			e = v
		case string:
			e = errors.New(v)
		default:
			if s, ok := i.(stringer); ok {
				e = errors.New(s.String())
			} else {
				e = fmt.Errorf("%v", i)
			}
		}
	}
	return
}

func writeStrings(buf *bytes.Buffer, strings []string) {
	for _, s := range strings {
		_, _ = buf.WriteString(s)
	}
}

func startPipe(r *Result, pipeTask func(v any) *Future, pipePromise *promise) {
	if pipeTask != nil {
		f := pipeTask(r.Result)
		f.OnSuccess(func(v any) {
			_ = pipePromise.Resolve(v)
		}).OnFailure(func(v any) {
			_ = pipePromise.Reject(getError(v))
		})
	}
}

func getFutureReturnVal(r *Result) (any, error) {
	switch r.Typ {
	case ResultSuccess:
		return r.Result, nil
	case ResultFailure:
		return nil, getError(r.Result)
	default:
		return nil, getError(r.Result) // &CancelledError{}
	}
}

func execCallback(
	r *Result,
	dones []func(v any),
	fails []func(v any),
	always []func(v any),
	cancels []func(),
) {
	if r.Typ == ResultCancelled {
		for _, f := range cancels {
			func() {
				defer func() {
					if e := recover(); e != nil {
						err := newErrorWithStacks(e)
						fmt.Println("error happens:\n ", err)
					}
				}()
				f()
			}()
		}
		return
	}

	var callbacks []func(v any)
	if r.Typ == ResultSuccess {
		callbacks = dones
	} else {
		callbacks = fails
	}

	forFs := func(s []func(v any)) {
		forSlice(s, func(f func(v any)) { f(r.Result) })
	}

	forFs(callbacks)
	forFs(always)
}

func forSlice(s []func(v any), f func(func(v any))) {
	for _, e := range s {
		func() {
			defer func() {
				if e := recover(); e != nil {
					err := newErrorWithStacks(e)
					fmt.Println("error happens:\n ", err)
				}
			}()
			f(e)
		}()
	}
}
