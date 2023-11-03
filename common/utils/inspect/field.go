// Inspired by github.com/chenzhuoyu/go-inspect

package inspect

import (
	"errors"
	"reflect"
	"unsafe"
)

type Field struct {
	t *rtype
	p unsafe.Pointer
}

// Get returns the value of the referenced field, even if it's private.
func (f Field) Get() reflect.Value {
	if f.t == nil {
		panic(errors.New("inspect: invalid field"))
	}
	return reflect.ValueOf(f.read().pack())
}

// Set updates the value of the referenced field, even if it's private.
func (f Field) Set(v reflect.Value) {
	if f.t == nil {
		panic(errors.New("inspect: invalid field"))
	}

	v = v.Convert(packType(f.t))
	typedmemmove(f.t, f.p, f.addr(v))
}

func (f Field) read() eface {
	if ifaceIndir(f.t) {
		return eface{_type: f.t, data: f.p}
	} else {
		return eface{_type: f.t, data: *(*unsafe.Pointer)(f.p)}
	}
}

func (f Field) addr(v reflect.Value) unsafe.Pointer {
	if ifaceIndir(f.t) {
		return (*eface)(unsafe.Pointer(&v)).data
	} else {
		return unsafe.Pointer(&((*eface)(unsafe.Pointer(&v)).data))
	}
}

func newField(v reflect.Value, fv reflect.StructField) Field {
	return Field{
		t: unpackType(fv.Type),
		p: unsafe.Pointer(uintptr((*eface)(unsafe.Pointer(&v)).data) + fv.Offset),
	}
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func derefValue(t reflect.Value) reflect.Value {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
