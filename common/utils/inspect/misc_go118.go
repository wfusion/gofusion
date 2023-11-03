//go:build go1.18
// +build go1.18

package inspect

import "github.com/pkg/errors"

func mustOk[T any](out T, ok bool) T {
	if !ok {
		panic(errors.Errorf("get %T with ok is false", out))
	}
	return out
}
