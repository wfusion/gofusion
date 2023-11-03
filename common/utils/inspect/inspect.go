// Inspired by github.com/chenzhuoyu/go-inspect

package inspect

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

var (
	lock    sync.RWMutex
	structs = make(map[string]reflect.Type)
	funcs   = make(map[string]unsafe.Pointer)

	supportTypes = map[reflect.Kind]bool{
		reflect.Bool:       true,
		reflect.Int:        true,
		reflect.Int8:       true,
		reflect.Int16:      true,
		reflect.Int32:      true,
		reflect.Int64:      true,
		reflect.Uint:       true,
		reflect.Uint8:      true,
		reflect.Uint16:     true,
		reflect.Uint32:     true,
		reflect.Uint64:     true,
		reflect.Uintptr:    true,
		reflect.Float32:    true,
		reflect.Float64:    true,
		reflect.Complex64:  true,
		reflect.Complex128: true,
		reflect.Array:      true,
		reflect.Chan:       true,
		reflect.Map:        true,
		reflect.Slice:      true,
		reflect.String:     true,
		reflect.Struct:     true,
		reflect.Interface:  true,
	}
)

// TypeOf find the type by package path and name
func TypeOf(typeName string) reflect.Type {
	lock.RLock()
	defer lock.RUnlock()
	if typ, ok := structs[typeName]; ok {
		return typ
	}
	return nil
}

// RuntimeTypeOf find the type by package path and name in runtime
func RuntimeTypeOf(typeName string) (r reflect.Type) {
	enumerateTypes(func(typ reflect.Type) bool {
		if typ.PkgPath()+"."+typ.Name() == typeName {
			r = typ
			return false
		}
		return true
	})
	return
}

// FuncOf find the function entry by package path and name,
// the function should be linked and should not be inlined
func FuncOf(funcName string) unsafe.Pointer {
	lock.RLock()
	defer lock.RUnlock()
	if entry, ok := funcs[funcName]; ok {
		return unsafe.Pointer(&entry)
	}
	return nil
}

// RuntimeFuncOf find the function entry by package path and name in runtime,
// the function should be linked and should not be inlined
func RuntimeFuncOf(funcName string) (r unsafe.Pointer) {
	enumerateFuncs(func(fn *runtime.Func, addr uintptr) bool {
		if fn.Name() == funcName {
			r = unsafe.Pointer(&addr)
			return false
		}
		return true
	})
	return
}

func init() {
	lock.Lock()
	defer lock.Unlock()

	// inspect structs
	enumerateTypes(func(typ reflect.Type) bool {
		pkgName := typ.PkgPath()
		typeName := typ.Name()
		if pkgName != "" && typeName != "" {
			structs[pkgName+"."+typeName] = typ
		}

		return true
	})

	// inspect function
	enumerateFuncs(func(fn *runtime.Func, addr uintptr) bool {
		if funcName := fn.Name(); funcName != "" {
			funcs[funcName] = unsafe.Pointer(&addr)
		}
		return true
	})
}
