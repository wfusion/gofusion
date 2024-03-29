package utils

func SliceRemove[T any, TS ~[]T](s TS, filter func(t T) bool) (d TS) {
	for i := len(s) - 1; i >= 0; i-- {
		if filter(s[i]) {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func SliceReverse[T any, TS ~[]T](s TS) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		opp := len(s) - 1 - i
		s[i], s[opp] = s[opp], s[i]
	}
}

// SliceSplit Separate objects into several size
func SliceSplit[T any, TS ~[]T](arr TS, size int) []TS {
	if size <= 0 {
		return []TS{arr}
	}

	chunkSet := make([]TS, 0, len(arr)/size+1)

	var chunk TS
	for len(arr) > size {
		chunk, arr = arr[:size], arr[size:]
		chunkSet = append(chunkSet, chunk)
	}
	if len(arr) > 0 {
		chunkSet = append(chunkSet, arr[:])
	}

	return chunkSet
}
