//go:build go1.16 && !go1.18
// +build go1.16,!go1.18

package inspect

import "github.com/pkg/errors"

func mustOk(out interface{}, ok bool) interface{} {
	if !ok {
		panic(errors.Errorf("get %T with ok is false", out))
	}
	return out
}
