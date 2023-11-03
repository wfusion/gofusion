//go:build go1.16 && !go1.18
// +build go1.16,!go1.18

// Inspired by github.com/chenzhuoyu/go-inspect

package inspect

import (
	"reflect"
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

func SetField(obj interface{}, fieldName string, val interface{}) {
	mustOk(FieldByName(derefValue(reflect.ValueOf(obj)), fieldName)).(Field).Set(reflect.ValueOf(val))
}

func GetField(obj interface{}, fieldName string) (r interface{}) {
	r = mustOk(FieldByName(derefValue(reflect.ValueOf(obj)), fieldName)).(Field).Get().Interface()
	return
}
