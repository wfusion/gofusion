package utils

func MapKeys[T comparable, K any](m map[T]K) (keys []T) {
	if m == nil {
		return
	}
	keys = make([]T, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return
}

func MapValues[T comparable, K any](m map[T]K) (vals []K) {
	if m == nil {
		return
	}
	vals = make([]K, 0, len(m))
	for _, val := range m {
		vals = append(vals, val)
	}
	return
}

func MapValuesByKeys[T comparable, K any, TS ~[]T](m map[T]K, keys TS) (vals []K) {
	if m == nil {
		return
	}
	vals = make([]K, 0, len(keys))
	for _, key := range keys {
		val, ok := m[key]
		if !ok {
			continue
		}
		vals = append(vals, val)
	}
	return
}

func MapMerge[T comparable, K any](a, b map[T]K) (r map[T]K) {
	r = make(map[T]K, len(a)+len(b))
	for k, v := range a {
		r[k] = v
	}
	for k, v := range b {
		r[k] = v
	}
	return
}

func MapSliceToMap[T comparable, K any](s []map[T]K) (d map[T]K) {
	d = make(map[T]K, len(s))
	for _, kv := range s {
		for k, v := range kv {
			d[k] = v
		}
	}
	return
}

func SliceToMap[K comparable, V any](s []V, groupFn func(v V) K) (d map[K]V) {
	d = make(map[K]V)
	for _, i := range s {
		d[groupFn(i)] = i
	}
	return
}
