//go:build go1.18
// +build go1.18

package inspect

import (
	"reflect"
	"runtime"
	"unsafe"
)

func enumerateTypes(cb func(reflect.Type) bool) {
	t0 := reflect.TypeOf(struct{}{})
	sections, typeLinks := typelinks()
	for i, typeLink := range typeLinks {
		for _, link := range typeLink {
			(*eface)(unsafe.Pointer(&t0)).data = resolveTypeOff(sections[i], link)
			if t0.Kind() != reflect.Ptr || !supportTypes[t0.Elem().Kind()] {
				continue
			}
			typ := t0.Elem()
			if typ.PkgPath() == "" || typ.Name() == "" {
				continue
			}
			if !cb(typ) {
				return
			}
		}
	}
}

func enumerateFuncs(cb func(*runtime.Func, uintptr) bool) {
	for _, md := range activeModules() {
		for _, tab := range md.ftab {
			f := tab // should not take from &tab.entry because go syntax
			absoluteAddr := textAddr(md, f.entry)
			if fn := runtime.FuncForPC(absoluteAddr); fn != nil {
				if !cb(fn, absoluteAddr) {
					return
				}
			}
		}
	}
}
