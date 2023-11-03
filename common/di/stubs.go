package di

import (
	"fmt"
	"reflect"

	_ "unsafe"
)

//go:linkname newParam go.uber.org/dig.newParam
func newParam(t reflect.Type, c containerStore) (param, error)

// param go.uber.org/dig.param
type param interface {
	fmt.Stringer
	Build(store containerStore) (reflect.Value, error)
	DotParam() []*struct{}
}

// containerStore go.uber.org/dig.containerStore
type containerStore interface {
}
