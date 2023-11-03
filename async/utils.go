package async

import (
	"fmt"
	"reflect"
)

func wrapParams(fn any) (argType reflect.Type, embed bool) {
	typ := reflect.TypeOf(fn)

	inLength := typ.NumIn()
	if inLength == 1 {
		return
	}
	if inLength == 2 {
		return typ.In(1), false
	}

	fields := make([]reflect.StructField, 0, inLength)
	for i := 1; i < inLength; i++ {
		fields = append(fields, reflect.StructField{
			Name:      fmt.Sprintf("Arg%X", i+1),
			PkgPath:   "",
			Type:      typ.In(i),
			Tag:       "",
			Offset:    0,
			Index:     nil,
			Anonymous: false,
		})
	}

	return reflect.StructOf(fields), true
}

func unwrapParams(typ reflect.Type, embed bool, arg any) (params []any) {
	if typ == nil {
		return
	}

	if !embed {
		return []any{arg}
	}

	argVal := reflect.Indirect(reflect.ValueOf(arg))
	num := argVal.Type().NumField()
	params = make([]any, 0, num)
	for i := 0; i < num; i++ {
		params = append(params, argVal.Field(i).Interface())
	}

	return
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
