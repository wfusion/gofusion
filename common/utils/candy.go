package utils

import (
	"io"
	"runtime/debug"

	"github.com/pkg/errors"
)

func Catch(fn any) (isPanic bool, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		isPanic = true
		switch v := r.(type) {
		case error:
			err = errors.Wrapf(v, "panic when call Catch function =>\n%s", debug.Stack())
		default:
			err = errors.Errorf("panic when call Catch function: %s =>\n%s", r, debug.Stack())
		}
	}()

	// check supported function
	var v any
	switch f := fn.(type) {
	case func():
		f()
	case func() error:
		err = f()
	case func() (any, error):
		v, err = f()
	default:
		panic(errors.Errorf("unsupported function signature %T", fn))
	}

	if err != nil {
		return
	}
	if ve, ok := v.(error); ok {
		err = ve
	}
	return
}

func CheckIfAny(fnList ...func() error) error {
	for _, fn := range fnList {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func IfAny(fnList ...func() bool) {
	for _, fn := range fnList {
		if fn() {
			break
		}
	}
}

func MustSuccess(err error) {
	if err != nil {
		panic(err)
	}
}

func Must[T any](out T, err error) T {
	if err != nil {
		panic(err)
	}
	return out
}

func MustOk[T any](out T, ok bool) T {
	if !ok {
		panic(errors.Errorf("get %T with ok is false", out))
	}
	return out
}

type closerA interface{ Close() }
type closerB[T any] interface{ Close() T }

func CloseAnyway[T any](closer T) {
	if any(closer) == nil {
		return
	}

	switch c := any(closer).(type) {
	case io.Closer:
		_ = c.Close()
	case closerA:
		c.Close()
	case closerB[T]:
		c.Close()
	}
}

type flusherA interface{ Flush() }
type flusherB interface{ Flush() error }

func FlushAnyway[T any](flusher T) {
	if any(flusher) == nil {
		return
	}

	switch f := any(flusher).(type) {
	case flusherA:
		f.Flush()
	case flusherB:
		_ = f.Flush()
	}
}

func ErrIgnore(src error, ignored ...error) (dst error) {
	for _, target := range ignored {
		if errors.Is(src, target) {
			return
		}
	}
	return src
}

type lookupByFuzzyKeywordFuncType[T any] interface {
	func(string) T |
		func(string) (T, bool) |
		func(string) (T, error)
}

func LookupByFuzzyKeyword[T any, F lookupByFuzzyKeywordFuncType[T]](lookup F, keyword string) (v T) {
	fn := WrapFunc1[T](lookup)
	for _, k := range FuzzyKeyword(keyword) {
		if v = fn(k); !IsBlank(v) {
			return
		}
	}
	return
}
