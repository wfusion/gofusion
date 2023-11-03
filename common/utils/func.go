package utils

import (
	"reflect"
	"regexp"
	"runtime"

	"github.com/gobwas/glob"
	"github.com/mitchellh/mapstructure"
)

func GetFuncName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func WrapFunc(fn any) func(...any) {
	return func(a ...any) {
		runVariadicFunc(fn, a...)
	}
}

func WrapFuncAny(fn any) func(...any) []any {
	return func(a ...any) (b []any) {
		ret := runVariadicFunc(fn, a...)
		b = make([]any, 0, len(b))
		for i := 0; i < len(ret); i++ {
			b = append(b, ret[i].Interface())
		}
		return
	}
}

func WrapFunc1[T any](fn any) func(...any) T {
	return func(a ...any) (t T) {
		ret := runVariadicFunc(fn, a...)
		return ParseVariadicFuncResult[T](ret, 0)
	}
}

// WrapFunc2 wrap a function with any number inputs and 2 generic type return,
// return nothing if function has 0 outputs
// return T1 if function only has 1 output
// return function first output and last output if function has more than 2 outputs
func WrapFunc2[T1, T2 any](fn any) func(...any) (T1, T2) {
	return func(a ...any) (t1 T1, t2 T2) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		return
	}
}

func WrapFunc3[T1, T2, T3 any](fn any) func(...any) (T1, T2, T3) {
	return func(a ...any) (t1 T1, t2 T2, t3 T3) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		t3 = ParseVariadicFuncResult[T3](ret, 2)
		return
	}
}

func WrapFunc4[T1, T2, T3, T4 any](fn any) func(...any) (T1, T2, T3, T4) {
	return func(a ...any) (t1 T1, t2 T2, t3 T3, t4 T4) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		t3 = ParseVariadicFuncResult[T3](ret, 2)
		t4 = ParseVariadicFuncResult[T4](ret, 3)
		return
	}
}

func WrapFunc5[T1, T2, T3, T4, T5 any](fn any) func(...any) (T1, T2, T3, T4, T5) {
	return func(a ...any) (t1 T1, t2 T2, t3 T3, t4 T4, t5 T5) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		t3 = ParseVariadicFuncResult[T3](ret, 2)
		t4 = ParseVariadicFuncResult[T4](ret, 3)
		t5 = ParseVariadicFuncResult[T5](ret, 4)
		return
	}
}

func WrapFunc6[T1, T2, T3, T4, T5, T6 any](fn any) func(...any) (T1, T2, T3, T4, T5, T6) {
	return func(a ...any) (t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		t3 = ParseVariadicFuncResult[T3](ret, 2)
		t4 = ParseVariadicFuncResult[T4](ret, 3)
		t5 = ParseVariadicFuncResult[T5](ret, 4)
		t6 = ParseVariadicFuncResult[T6](ret, 5)
		return
	}
}

func WrapFunc7[T1, T2, T3, T4, T5, T6, T7 any](fn any) func(...any) (T1, T2, T3, T4, T5, T6, T7) {
	return func(a ...any) (t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7) {
		ret := runVariadicFunc(fn, a...)
		t1 = ParseVariadicFuncResult[T1](ret, 0)
		t2 = ParseVariadicFuncResult[T2](ret, 1)
		t3 = ParseVariadicFuncResult[T3](ret, 2)
		t4 = ParseVariadicFuncResult[T4](ret, 3)
		t5 = ParseVariadicFuncResult[T5](ret, 4)
		t6 = ParseVariadicFuncResult[T6](ret, 5)
		t7 = ParseVariadicFuncResult[T7](ret, 6)
		return
	}
}

func runVariadicFunc(fn any, a ...any) []reflect.Value {
	var (
		variadic   []reflect.Value
		typ        = reflect.TypeOf(fn)
		val        = reflect.ValueOf(fn)
		numIn      = typ.NumIn()
		isVariadic = typ.IsVariadic()
	)

	if isVariadic {
		b := a[numIn-1:]
		bt := typ.In(numIn - 1).Elem()
		variadic = make([]reflect.Value, 0, len(b))
		for _, param := range b {
			paramVal := reflect.ValueOf(param)
			if paramVal.CanConvert(bt) {
				variadic = append(variadic, paramVal.Convert(bt))
			} else {
				bo := reflect.New(bt).Elem().Interface()
				MustSuccess(mapstructure.Decode(param, &bo))
				variadic = append(variadic, reflect.ValueOf(bo))
			}
		}
		a = a[:numIn-1]
	}

	in := make([]reflect.Value, 0, len(a)+len(variadic))
	for idx, param := range a {
		pt := typ.In(idx)
		paramVal := reflect.ValueOf(param)
		if paramVal.CanConvert(pt) {
			in = append(in, paramVal.Convert(pt))
		} else {
			po := reflect.New(pt).Elem().Interface()
			MustSuccess(mapstructure.Decode(param, &po))
			in = append(in, reflect.ValueOf(po))
		}
	}
	in = append(in, variadic...)

	return val.Call(in)
}

func ParseVariadicFuncResult[T any](rs []reflect.Value, idx int) (t T) {
	var (
		ok  bool
		typ = reflect.TypeOf(t)
	)
	for i := idx; i < len(rs); i++ {
		r := rs[i]
		if !r.IsValid() || r.Type() == nil {
			continue
		}
		if v := r.Interface(); v != nil {
			if t, ok = v.(T); !ok && typ != nil && reflect.TypeOf(v).ConvertibleTo(typ) {
				t = r.Convert(typ).Interface().(T)
			}
		}
	}
	return
}

type getCallerOption struct {
	skipRegList        []*regexp.Regexp
	skipGlobList       []glob.Glob
	minimumCallerDepth int
}

func SkipRegexps(patterns ...string) OptionFunc[getCallerOption] {
	return func(o *getCallerOption) {
		for _, pattern := range patterns {
			o.skipRegList = append(o.skipRegList, regexp.MustCompile(pattern))
		}
	}
}

func SkipGlobs(patterns ...string) OptionFunc[getCallerOption] {
	return func(o *getCallerOption) {
		for _, pattern := range patterns {
			o.skipGlobList = append(o.skipGlobList, glob.MustCompile(pattern))
		}
	}
}

func SkipKnownDepth(minimumCallerDepth int) OptionFunc[getCallerOption] {
	return func(o *getCallerOption) {
		o.minimumCallerDepth = minimumCallerDepth
	}
}

// GetCaller retrieves the name after stack skip
func GetCaller(maximumCallerDepth int, opts ...OptionExtender) (frame *runtime.Frame) {
	opt := ApplyOptions[getCallerOption](opts...)
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(opt.minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])
outer:
	for f, hasMore := frames.Next(); hasMore; f, hasMore = frames.Next() {
		frame = &f

		// If the caller isn't part of this package, we're done
		for _, skipGlob := range opt.skipGlobList {
			if skipGlob.Match(f.File) {
				continue outer
			}
		}
		for _, s := range opt.skipRegList {
			if s.MatchString(f.File) {
				continue outer
			}
		}
		break
	}

	return
}
