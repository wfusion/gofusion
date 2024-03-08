package utils

func IsChannelClosed[T any](ch <-chan T) (data T, ok bool) {
	select {
	case d, opened := <-ch:
		if !opened {
			ok = true
		}
		data = d
		return
	default:
		return
	}
}
