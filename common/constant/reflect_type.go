package constant

import (
	"context"
	"reflect"

	"github.com/gin-gonic/gin"
)

var (
	ErrorType             = reflect.TypeOf((*error)(nil)).Elem()
	ContextType           = reflect.TypeOf((*context.Context)(nil)).Elem()
	GinContextType        = reflect.TypeOf((*gin.Context)(nil))
	AnyType               = reflect.TypeOf((*any)(nil)).Elem()
	AnySliceType          = reflect.TypeOf(([]any)(nil))
	MapStringAnySliceType = reflect.TypeOf(([]map[string]any)(nil))
	IntType               = reflect.TypeOf(int(0))
	IntSliceType          = reflect.TypeOf([]int(nil))
	UintType              = reflect.TypeOf(uint(0))
	UintSliceType         = reflect.TypeOf([]uint(nil))
	StringType            = reflect.TypeOf(string(""))
	StringSliceType       = reflect.TypeOf([]string(nil))
	Float32Type           = reflect.TypeOf(float32(0))
	Float64Type           = reflect.TypeOf(float64(0))
	BoolType              = reflect.TypeOf(bool(false))
)
