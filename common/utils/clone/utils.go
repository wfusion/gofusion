package clone

import "reflect"

func indirectType(s reflect.Type) (d reflect.Type) {
	if s == nil {
		return s
	}
	d = s
	for d.Kind() == reflect.Ptr {
		d = d.Elem()
	}
	return
}
