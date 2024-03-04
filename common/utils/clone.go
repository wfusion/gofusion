package utils

type clonable[T any] interface {
	Clone() T
}
