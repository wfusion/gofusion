package utils

type OptionExtender interface {
	applyOption(t any)
}

type OptionFunc[T any] func(*T)

func (o OptionFunc[T]) applyOption(a any) {
	if t, ok := a.(*T); ok {
		o(t)
	}
}

func ApplyOptions[T any](opts ...OptionExtender) (t *T) {
	t = new(T)
	for _, optional := range opts {
		if optional != nil {
			optional.applyOption(t)
		}
	}
	return
}
