package utils

func IsChannelClosed[T any](ch <-chan T) (data T, ok bool) {
	select {
	case d, closed := <-ch:
		if closed {
			ok = true
			return
		}
		return d, false
	default:
		return
	}
}
