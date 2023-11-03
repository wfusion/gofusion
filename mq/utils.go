package mq

import (
	"reflect"
)

func wrapParams(fn any) (argType reflect.Type) {
	typ := reflect.TypeOf(fn)

	inLength := typ.NumIn()
	if inLength == 1 {
		return
	}
	if inLength >= 2 {
		return typ.In(1)
	}

	return
}

func unwrapParams(typ reflect.Type, arg any) (params []any) {
	if typ == nil {
		return
	}

	return []any{arg}
}

func setParams(typ reflect.Type, embed bool, params ...any) (arg any) {
	if typ == nil {
		return
	}

	argValPtr := reflect.New(typ)
	argVal := argValPtr.Elem()
	if !embed {
		if len(params) > 0 {
			argVal.Set(reflect.ValueOf(params[0]))
		}
		return argValPtr.Interface()
	}

	for i := 0; i < len(params); i++ {
		ft := argVal.Field(i)
		ft.Set(reflect.ValueOf(params[i]).Convert(ft.Type()))
	}
	return argValPtr.Interface()
}
