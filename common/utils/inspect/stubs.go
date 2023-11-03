package inspect

import (
	"reflect"
	"unsafe"
)

//go:noescape
//go:linkname typelinks reflect.typelinks
func typelinks() ([]unsafe.Pointer, [][]int32)

//go:noescape
//go:linkname typedmemmove reflect.typedmemmove
func typedmemmove(_ *rtype, _ unsafe.Pointer, _ unsafe.Pointer)

//go:noescape
//go:linkname resolveTypeOff reflect.resolveTypeOff
func resolveTypeOff(unsafe.Pointer, int32) unsafe.Pointer

//go:noescape
//go:linkname activeModules runtime.activeModules
func activeModules() []*_moduleData

//go:linkname ifaceIndir reflect.ifaceIndir
func ifaceIndir(t *rtype) bool

//go:noescape
//go:linkname valueInterface reflect.valueInterface
func valueInterface(v reflect.Value, safe bool) any
