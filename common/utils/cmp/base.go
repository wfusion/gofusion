package cmp

import (
	"fmt"
	"reflect"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/utils"
)

func ComparablePtr[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return *a == *b
}
func SliceComparable[T comparable, TS ~[]T](a, b TS) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func SliceComparablePtr[T comparable, TS ~[]*T](a, b TS) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if deref(a[i]) != deref(b[i]) {
			return false
		}
	}
	return true
}

func TimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Equal(*b) && TimeLocationPtr(a.Location(), b.Location())
}

func TimeLocationPtr(a, b *time.Location) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.String() == b.String()
}

func DecimalPtr(a, b *decimal.Decimal) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Equal(*b)
}

func GormModelPtr(a, b *gorm.Model) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.ID == b.ID &&
		TimePtr(&a.CreatedAt, &b.CreatedAt) &&
		TimePtr(&a.UpdatedAt, &b.UpdatedAt) &&
		a.DeletedAt.Valid == b.DeletedAt.Valid &&
		TimePtr(&a.DeletedAt.Time, &b.DeletedAt.Time)
}

type _comparable[T any] interface {
	Equals(other T) bool
}

func Slice[T _comparable[T], TS ~[]T](a, b TS, sortFn func(i, j T) int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}

	if sortFn != nil {
		utils.SortStable(a, sortFn)
		utils.SortStable(b, sortFn)
	}

	for i := 0; i < len(a); i++ {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

func SliceAny[T any, TS ~[]T](a, b TS, sortFn func(i, j T) int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}

	if sortFn != nil {
		utils.SortStable(a, sortFn)
		utils.SortStable(b, sortFn)
	}

	for i := 0; i < len(a); i++ {
		if !anything(a[i], b[i]) {
			return false
		}
	}
	return true
}

func Map[T _comparable[T], K comparable](a, b map[K]T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}

	for ak, av := range a {
		bv, ok := b[ak]
		if !ok || !av.Equals(bv) {
			return false
		}
	}

	return true
}

func MapAny[K comparable, T any](a, b map[K]T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}

	for ak, av := range a {
		bv, ok := b[ak]
		if !ok || !anything(av, bv) {
			return false
		}

	}
	return true
}

func anything(a, b any) bool {
	switch av := a.(type) {
	case
			bool,
			string, uintptr,
			int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64,
			complex64, complex128:
		return a == b
	case
			*bool,
			*string, *uintptr,
			*int, *int8, *int16, *int32, *int64,
			*uint, *uint8, *uint16, *uint32, *uint64,
			*float32, *float64,
			*complex64, *complex128:
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return anything(deref(a), deref(b))
	case decimal.Decimal:
		bv := b.(decimal.Decimal)
		return DecimalPtr(&av, &bv)
	case *decimal.Decimal:
		bv := b.(*decimal.Decimal)
		return DecimalPtr(av, bv)
	case time.Time:
		bv := b.(time.Time)
		return TimePtr(&av, &bv)
	case *time.Time:
		bv := b.(*time.Time)
		return TimePtr(av, bv)
	case time.Location:
		bv := b.(time.Location)
		return TimeLocationPtr(&av, &bv)
	case *time.Location:
		bv := b.(*time.Location)
		return TimeLocationPtr(av, bv)
	case []bool:
		return SliceComparable(a.([]bool), b.([]bool))
	case []string:
		return SliceComparable(a.([]string), b.([]string))
	case []uintptr:
		return SliceComparable(a.([]uintptr), b.([]uintptr))
	case []int:
		return SliceComparable(a.([]int), b.([]int))
	case []int8:
		return SliceComparable(a.([]int8), b.([]int8))
	case []int16:
		return SliceComparable(a.([]int16), b.([]int16))
	case []int32:
		return SliceComparable(a.([]int32), b.([]int32))
	case []int64:
		return SliceComparable(a.([]int64), b.([]int64))
	case []uint:
		return SliceComparable(a.([]uint), b.([]uint))
	case []uint8:
		return SliceComparable(a.([]uint8), b.([]uint8))
	case []uint16:
		return SliceComparable(a.([]uint16), b.([]uint16))
	case []uint32:
		return SliceComparable(a.([]uint32), b.([]uint32))
	case []uint64:
		return SliceComparable(a.([]uint64), b.([]uint64))
	case []float32:
		return SliceComparable(a.([]float32), b.([]float32))
	case []float64:
		return SliceComparable(a.([]float64), b.([]float64))
	case []complex64:
		return SliceComparable(a.([]complex64), b.([]complex64))
	case []complex128:
		return SliceComparable(a.([]complex128), b.([]complex128))
	case []any:
		return SliceAny(a.([]any), b.([]any), nil)
	case []map[string]any:
		return SliceAny(a.([]map[string]any), b.([]map[string]any), nil)
	case map[string]any:
		return MapAny(av, b.(map[string]any))
	default:
		return reflect.DeepEqual(a, b)
	}
}

func deref(p any) (v any) {
	switch pp := p.(type) {
	case *bool:
		v = *pp
	case *string:
		v = *pp
	case *int:
		v = *pp
	case *int8:
		v = *pp
	case *int16:
		v = *pp
	case *int32:
		v = *pp
	case *int64:
		v = *pp
	case *uint:
		v = *pp
	case *uint8:
		v = *pp
	case *uint16:
		v = *pp
	case *uint32:
		v = *pp
	case *uint64:
		v = *pp
	case *float32:
		v = *pp
	case *float64:
		v = *pp
	case *complex64:
		v = *pp
	case *complex128:
		v = *pp
	case *uintptr:
		v = *pp
	case *[]byte:
		v = *pp
	case *any:
		v = *pp
	default:
		panic(fmt.Errorf("unsupported type %T", pp))
	}
	return
}
