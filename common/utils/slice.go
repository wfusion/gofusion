package utils

func SliceRemove[T any, TS ~[]T](s TS, filter func(t T) bool) (d TS) {
	for i, item := range s {
		if filter(item) {
			s = append(s[i:], s[i+1:]...)
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
