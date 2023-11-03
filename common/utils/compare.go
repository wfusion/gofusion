package utils

import (
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/constraint"
)

var (
	ErrEmptyArray = errors.New("empty array")
)

func Max[T constraint.Sortable](arr ...T) T {
	if len(arr) == 0 {
		panic(ErrEmptyArray)
	}

	max := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i] > max {
			max = arr[i]
		}
	}

	return max
}

func Min[T constraint.Sortable](arr ...T) T {
	if len(arr) == 0 {
		panic(ErrEmptyArray)
	}

	min := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i] < min {
			min = arr[i]
		}
	}

	return min
}

func IsInRange[T constraint.Sortable](num, min, max T) bool {
	if num < min || num > max {
		return false
	}
	return true
}
