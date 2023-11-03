package inspect

import (
	"reflect"
	"unsafe"
)

type tflag uint8
type nameOff int32 // offset to a name
type typeOff int32 // offset to an *rtype
type textOff int32 // offset from top of text section

// iface runtime.iface
type iface struct {
	tab  *itab
	data unsafe.Pointer
}

// eface runtime.eface
type eface struct {
	_type *rtype
	data  unsafe.Pointer
}

func (e eface) pack() (r interface{}) { *(*eface)(unsafe.Pointer(&r)) = e; return }

// rtype reflect.rtype, declare in internal/api.Type after go1.21
type rtype struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      tflag   // extra type information flags, reflect.tflag
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte   // garbage collection data
	str       nameOff // string form
	ptrToThis typeOff // type for pointer to this type, may be zero
}

type itab struct {
	inter *interfaceType
	_type *rtype
	hash  uint32 // copy of _type.hash. Used for type switches.
	_     [4]byte
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}

type interfaceType struct {
	typ     rtype
	pkgpath name
	mhdr    []iMethod
}

type name struct {
	bytes *byte
}

type iMethod struct {
	name nameOff
	typ  typeOff
}

var (
	itabRtype = func(v interface{}) *itab {
		t := reflect.TypeOf(v).Elem()
		return (*iface)(unsafe.Pointer(&t)).tab
	}(new(reflect.Type))
)

func unpackType(t reflect.Type) *rtype {
	return (*rtype)((*eface)(unsafe.Pointer(&t)).data)
}

func packType(t *rtype) (r reflect.Type) {
	(*iface)(unsafe.Pointer(&r)).tab = itabRtype
	(*iface)(unsafe.Pointer(&r)).data = unsafe.Pointer(t)
	return
}
