package utils

import (
	"math"
	"strconv"
	"strings"
)

func DecimalPlaces(num float64) int {
	str := strconv.FormatFloat(num, 'f', -1, 64)
	if i := strings.LastIndex(str, "."); i >= 0 {
		return len(str[i+1:])
	}
	return 0
}

func IntNarrow(num int) (result any) {
	IfAny(
		func() bool { result = int8(num); return IsInRange(num, math.MinInt8, math.MaxInt8) },
		func() bool { result = int16(num); return IsInRange(num, math.MinInt16, math.MaxInt16) },
		func() bool { result = int32(num); return IsInRange(num, math.MinInt32, math.MaxInt32) },
		func() bool { result = int64(num); return IsInRange(num, math.MinInt64, math.MaxInt64) },
	)
	return
}

func UintNarrow(num uint) (result any) {
	IfAny(
		func() bool { result = uint8(num); return IsInRange(num, 0, math.MaxUint8) },
		func() bool { result = uint16(num); return IsInRange(num, 0, math.MaxUint16) },
		func() bool { result = uint32(num); return IsInRange(num, 0, math.MaxUint32) },
		func() bool { result = uint64(num); return IsInRange(num, 0, math.MaxUint64) },
	)
	return
}
