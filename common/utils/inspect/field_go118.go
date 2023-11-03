//go:build go1.18
// +build go1.18

// Inspired by github.com/chenzhuoyu/go-inspect

package inspect

import (
	"reflect"
	"unsafe"
)

// FieldByName locates a field with name.
func FieldByName(v reflect.Value, name string) (Field, bool) {
	if fv, ok := v.Type().FieldByName(name); !ok {
		return Field{}, false
	} else {
		return newField(v, fv), true
	}
}

// FieldAt locates a field with index.
func FieldAt(v reflect.Value, idx int) (Field, bool) {
	if idx < 0 || idx >= v.NumField() {
		return Field{}, false
	} else {
		return newField(v, v.Type().Field(idx)), true
	}
}

func SetField[T any](obj any, fieldName string, val T) {
	mustOk(FieldByName(derefValue(reflect.ValueOf(obj)), fieldName)).Set(reflect.ValueOf(val))
}

func GetField[T any](obj any, fieldName string) (r T) {
	r, _ = mustOk(FieldByName(derefValue(reflect.ValueOf(obj)), fieldName)).Get().Interface().(T)
	return
}

// deprecated
func setField[T any](obj any, fieldName string, val T) {
	v := reflect.ValueOf(obj)
	t := reflect.Indirect(v).Type()
	*(*T)(unsafe.Pointer(v.Pointer() + mustOk(t.FieldByName(fieldName)).Offset)) = val
}

// deprecated
func getField[T any](obj any, fieldName string) (r T) {
	v := reflect.Indirect(reflect.ValueOf(obj))
	r, _ = valueInterface(v.FieldByName(fieldName), false).(T)
	return
}
