package utils

import (
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/constraint"
)

// SliceMapping Mapping slice convert go1.18 version
func SliceMapping[T, K any](s []T, mapFn func(t T) K) (d []K) {
	if s == nil {
		return
	}
	d = make([]K, 0, len(s))
	for _, item := range s {
		d = append(d, mapFn(item))
	}
	return
}

// SortableToGeneric convert sortable type to generic type
func SortableToGeneric[T, K constraint.Sortable](s T) (d K) {
	switch any(d).(type) {
	case int:
		return any(cast.ToInt(s)).(K)
	case int8:
		return any(cast.ToInt8(s)).(K)
	case int16:
		return any(cast.ToInt16(s)).(K)
	case int32:
		return any(cast.ToInt32(s)).(K)
	case int64:
		return any(cast.ToInt64(s)).(K)
	case *int:
		return any(AnyPtr(cast.ToInt(s))).(K)
	case *int8:
		return any(AnyPtr(cast.ToInt8(s))).(K)
	case *int16:
		return any(AnyPtr(cast.ToInt16(s))).(K)
	case *int32:
		return any(AnyPtr(cast.ToInt32(s))).(K)
	case *int64:
		return any(AnyPtr(cast.ToInt64(s))).(K)
	case uint:
		return any(cast.ToUint(s)).(K)
	case uint8:
		return any(cast.ToUint8(s)).(K)
	case uint16:
		return any(cast.ToUint16(s)).(K)
	case uint32:
		return any(cast.ToUint32(s)).(K)
	case uint64:
		return any(cast.ToUint64(s)).(K)
	case *uint:
		return any(AnyPtr(cast.ToUint(s))).(K)
	case *uint8:
		return any(AnyPtr(cast.ToUint8(s))).(K)
	case *uint16:
		return any(AnyPtr(cast.ToUint16(s))).(K)
	case *uint32:
		return any(AnyPtr(cast.ToUint32(s))).(K)
	case *uint64:
		return any(AnyPtr(cast.ToUint64(s))).(K)
	case float32:
		return any(cast.ToFloat32(s)).(K)
	case float64:
		return any(cast.ToFloat64(s)).(K)
	case *float32:
		return any(AnyPtr(cast.ToFloat32(s))).(K)
	case *float64:
		return any(AnyPtr(cast.ToFloat64(s))).(K)
	case string:
		return any(cast.ToString(s)).(K)
	case *string:
		return any(AnyPtr(cast.ToString(s))).(K)
	default:
		panic(errors.Errorf("cannot mapping %T", d))
	}
}

var sortableReflectType = []reflect.Type{
	constant.IntType,
	constant.UintType,
	constant.StringType,
	constant.Float32Type,
	constant.Float64Type,
	constant.BoolType,
}

// ComparableToSortable convert generic type to sortable type
func ComparableToSortable[T comparable](s T) (d any) {
	val := reflect.ValueOf(s)
	typ := val.Type()
	for _, sortableType := range sortableReflectType {
		if typ.ConvertibleTo(sortableType) {
			return val.Convert(sortableType).Interface()
		}
	}

	return
}

// SliceConvert >= go1.18 recommend to use SliceMapping
func SliceConvert(src any, dstType reflect.Type) any {
	srcVal := reflect.ValueOf(src)
	srcType := reflect.TypeOf(src)
	dstVal := reflect.Indirect(reflect.New(dstType))
	if srcType.Kind() != reflect.Slice || dstType.Kind() != reflect.Slice {
		panic(errors.Errorf("src or dst type is invalid [src[%s] dst[%s]]", srcType.Kind(), dstType.Kind()))
	}

	isInterfaceSlice := false
	srcElemType := srcType.Elem()
	if srcType == constant.AnySliceType {
		if srcVal.Len() == 0 {
			return dstVal.Interface()
		}
		srcElemType = reflect.TypeOf(srcVal.Index(0).Interface())
		isInterfaceSlice = true
	}

	dstElemType := dstType.Elem()
	if !srcElemType.ConvertibleTo(dstElemType) {
		panic(errors.Errorf("src elem is not convertible to dst elem [src[%s] dst[%s]]",
			srcElemType.Kind(), dstElemType.Kind()))
	}

	length := srcVal.Len()
	for i := 0; i < length; i++ {
		srcElem := srcVal.Index(i)
		if isInterfaceSlice {
			srcElem = reflect.ValueOf(srcVal.Index(i).Interface())
		}
		dstVal = reflect.Append(dstVal, srcElem.Convert(dstElemType))
	}

	return dstVal.Interface()
}

func AnyPtr[T any](s T) *T { return &s }

// UnsafeStringToBytes converts string to byte slice without a memory allocation.
// Fork from github.com/gin-gonic/gin@v1.7.7/internal/bytesconv/bytesconv.go
func UnsafeStringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// UnsafeBytesToString converts byte slice to string without a memory allocation.
// Fork from github.com/gin-gonic/gin@v1.7.7/internal/bytesconv/bytesconv.go
func UnsafeBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
