package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Enumerable[T comparable, TS ~[]T] interface {
	Enum(s string) TS
	String(k any) string
	IsValid(t T) bool
}

type enumStringOption struct {
	ignoreCaseSensitivity bool
}

type enumStringOptFn func(*enumStringOption)

func IgnoreEnumStringCase() enumStringOptFn {
	return func(e *enumStringOption) {
		e.ignoreCaseSensitivity = true
	}
}

func NewEnumString[T comparable, TS ~[]T](mapping map[T]string, opts ...OptionExtender) Enumerable[T, TS] {
	opt := ApplyOptions[enumStringOption](opts...)
	if len(mapping) == 0 {
		panic(errors.New("enum mapping is empty"))
	}
	return (&enumString[T, TS]{
		mapping:    mapping,
		ignoreCase: opt.ignoreCaseSensitivity,
	}).init()
}

type enumString[T comparable, TS ~[]T] struct {
	ignoreCase      bool
	elemType        reflect.Type
	elemSliceType   reflect.Type
	prefix          string
	mapping         map[T]string
	reversedMapping map[string]TS
}

func (e *enumString[T, TS]) Enum(s string) TS {
	s = e.caseSensitivityConv(s)
	if v, ok := e.reversedMapping[s]; ok {
		return TS(SliceConvert(v, e.elemSliceType))
	}
	return nil
}

func (e *enumString[T, TS]) String(k any) string {
	if reflect.TypeOf(k).ConvertibleTo(e.elemType) {
		k = reflect.ValueOf(k).Convert(e.elemType).Interface().(T)
	}
	if t, ok := k.(T); !ok {
		return fmt.Sprintf("%s(%v)", e.prefix, k)
	} else {
		if v, ok := e.mapping[t]; ok {
			return v
		}
		// avoid stack overflow for Stringer implement
		sortable := ComparableToSortable(t)
		if sortable != nil {
			return fmt.Sprintf("%s(%+v)", e.prefix, sortable)
		} else {
			return fmt.Sprintf("%s(N/A)", e.prefix)
		}
	}
}

func (e *enumString[T, TS]) IsValid(t T) bool {
	_, ok := e.mapping[t]
	return ok
}

func (e *enumString[T, TS]) init() Enumerable[T, TS] {
	// get key
	var key any
	for k := range e.mapping {
		key = k
		break
	}
	e.elemType = reflect.TypeOf(key)
	e.elemSliceType = reflect.SliceOf(e.elemType)

	// get prefix name
	e.prefix = cases.Title(language.English, cases.NoLower).String(e.elemType.Name())

	// get reversed mapping
	e.reversedMapping = make(map[string]TS, len(e.mapping))
	for k, v := range e.mapping {
		v = e.caseSensitivityConv(v)
		e.reversedMapping[v] = append(e.reversedMapping[v], k)
	}

	return e
}

func (e *enumString[T, TS]) caseSensitivityConv(s string) string {
	if e.ignoreCase {
		return strings.ToLower(s)
	}
	return s
}
