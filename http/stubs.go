package http

import (
	"reflect"

	_ "unsafe"
)

//go:noescape
//go:linkname valueInterface reflect.valueInterface
func valueInterface(v reflect.Value, safe bool) any
