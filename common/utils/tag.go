package utils

import (
	"reflect"
)

type parseTagOption struct {
	tag           string
	overwrite     bool
	unmarshalType unmarshalType
}

func ParseTagName(tag string) OptionFunc[parseTagOption] {
	return func(o *parseTagOption) {
		o.tag = tag
	}
}

func ParseTagOverwrite(overwrite bool) OptionFunc[parseTagOption] {
	return func(o *parseTagOption) {
		o.overwrite = overwrite
	}
}

func ParseTagUnmarshalType(unmarshalTag unmarshalType) OptionFunc[parseTagOption] {
	return func(o *parseTagOption) {
		o.unmarshalType = unmarshalTag
	}
}

func ParseTag(data any, opts ...OptionExtender) (err error) {
	opt := ApplyOptions[parseTagOption](opts...)
	stepInKinds := NewSet(reflect.Struct, reflect.Array, reflect.Slice, reflect.Map)
	TraverseValue(data, false, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() || !value.CanSet() || !value.CanAddr() || !value.CanInterface() {
			return
		}

		vk := value.Kind()
		stepIn = stepInKinds.Contains(vk) ||
			(vk == reflect.Ptr && value.Elem().IsValid() && value.Elem().Kind() == reflect.Struct)

		defaultString := field.Tag.Get(opt.tag)
		if IsStrBlank(defaultString) || (!opt.overwrite && !IsBlank(value)) {
			return
		}

		defaultValue := reflect.New(value.Type()).Interface()
		if err = Unmarshal(defaultString, defaultValue, opt.unmarshalType); err != nil {
			end = true
			return
		}
		value.Set(reflect.ValueOf(defaultValue).Elem())
		return
	})

	return
}
