package utils

import (
	"container/list"
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm/schema"
)

// IsBlank gets whether the specified object is considered empty or not.
// fork from: github.com/stretchr/testify@v1.8.0/assert/assertions.go
func IsBlank(object any) bool {
	// get nil case out of the way
	if object == nil {
		return true
	}

	objVal, ok := object.(reflect.Value)
	if !ok {
		objVal = reflect.ValueOf(object)
	}
	switch objVal.Kind() {
	// collection types are empty when they have no element
	case reflect.Chan, reflect.Map, reflect.Slice:
		return objVal.Len() == 0
	// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objVal.IsNil() {
			return true
		}
		deref := objVal.Elem().Interface()
		return IsBlank(deref)
	// for all other types, compare against the zero value
	// array types are empty when they match their zero-initialized state
	default:
		zero := reflect.Zero(objVal.Type())
		return reflect.DeepEqual(objVal.Interface(), zero.Interface())
	}
}

func TraverseValue(data any, indirect bool, handler func(reflect.StructField, reflect.Value) (end, stepIn bool)) {
	v, ok := data.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(data)
	}
	v = IndirectValue(v)
	l := list.New()
	l.PushBack(v)
TraverseStruct:
	for l.Len() > 0 {
		e := IndirectValue(l.Remove(l.Front()).(reflect.Value))
		if !e.IsValid() {
			continue
		}
		t := IndirectType(e.Type())
		switch e.Kind() {
		case reflect.Array, reflect.Slice:
			for i, num := 0, e.Len(); i < num; i++ {
				l.PushBack(e.Index(i))
			}
		case reflect.Map:
			for iter := e.MapRange(); iter.Next(); {
				l.PushBack(iter.Key())
				l.PushBack(iter.Value())
			}
		case reflect.Struct:
			for i, num := 0, e.NumField(); i < num; i++ {
				ff := t.Field(i)
				fv := e.Field(i)
				if !fv.IsValid() {
					continue
				}
				if indirect {
					fv = IndirectValue(fv)
					ff.Type = IndirectType(ff.Type)
				}
				end, stepIn := handler(ff, fv)
				if end {
					break TraverseStruct
				}
				if stepIn {
					l.PushBack(fv)
				}
			}
		default:
			// do nothing
		}
	}
}

func GetFieldByTag(data any, tag, key string) (r reflect.Value, e error) {
	TraverseValue(data, true, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() {
			return false, false
		}
		if value.Type().Kind() == reflect.Struct {
			return false, true
		}
		tagV := field.Tag.Get(tag)
		if tagV == key {
			r = value
			end = true
			return
		}
		return
	})
	return
}

func GetFieldByTagWithKeys(data any, tag string, keys []string) (r reflect.Value, e error) {
	keySet := NewSet[string](keys...)
	TraverseValue(data, true, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() {
			return false, false
		}
		if value.Type().Kind() == reflect.Struct {
			return false, true
		}
		if keySet.Contains(field.Tag.Get(tag)) {
			r = value
			end = true
			return
		}
		return
	})
	return
}

func GetFieldTagValue(data any, tag string, pattern *regexp.Regexp) (tagValue string, e error) {
	TraverseValue(data, true, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() {
			return false, false
		}
		if value.Type().Kind() == reflect.Struct {
			return false, true
		}
		tagV := field.Tag.Get(tag)
		if pattern.Match([]byte(tagV)) {
			tagValue = tagV
			end = true
			return
		}
		return
	})
	return
}

func GetGormColumnValue(data any, column string) (columnVal reflect.Value, ok bool) {
	tagKey := strings.ToUpper(column)
	TraverseValue(data, true, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() {
			return false, false
		}
		if value.Type().Kind() == reflect.Struct {
			return false, true
		}
		tagSetting := schema.ParseTagSetting(field.Tag.Get("gorm"), ";")
		if _, ok := tagSetting[tagKey]; ok || tagSetting["COLUMN"] == column {
			columnVal = value
			end = true
			return
		}
		return
	})
	return
}

// EmbedsType Returns true if t embeds e or if any of the types embedded by t embed e.
// Forked from go.uber.org/dig@v1.16.1/inout.embedsType
func EmbedsType(i any, e reflect.Type) bool {
	// given `type A foo { *In }`, this function would return false for
	// embedding dig.In, which makes for some extra error checking in places
	// that call this function. Might be worthwhile to consider reflect.Indirect
	// usage to clean up the callers.

	if i == nil {
		return false
	}

	// maybe it's already a reflect.Type
	t, ok := i.(reflect.Type)
	if !ok {
		// take the type if it's not
		t = IndirectType(reflect.TypeOf(i))
	}

	// We are going to do a breadth-first search of all embedded fields.
	types := list.New()
	types.PushBack(t)
	for types.Len() > 0 {
		t := types.Remove(types.Front()).(reflect.Type)

		if t == e {
			return true
		}

		if t.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				types.PushBack(f.Type)
			}
		}
	}

	return false
}

func IndirectValue(s reflect.Value) (d reflect.Value) {
	if !s.IsValid() {
		return s
	}
	d = s
	for d.Kind() == reflect.Ptr {
		d = d.Elem()
	}
	return
}

func IndirectType(s reflect.Type) (d reflect.Type) {
	if s == nil {
		return s
	}
	d = s
	for d.Kind() == reflect.Ptr {
		d = d.Elem()
	}
	return
}
