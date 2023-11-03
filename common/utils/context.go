package utils

import (
	"context"
	"reflect"
)

// GetCtxAny with a default value
func GetCtxAny[T any](ctx context.Context, key string, args ...T) (val T) {
	if v := ctx.Value(key); v != nil {
		return v.(T)
	}
	if len(args) == 0 {
		return
	}
	return args[0]
}

// SetCtxAny with any value
func SetCtxAny[T any](ctx context.Context, key string, val T) context.Context {
	return context.WithValue(ctx, key, val)
}

// TravelCtx context parent traversal
func TravelCtx(child context.Context, fn func(ctx context.Context) bool) {
	v := reflect.ValueOf(child)
	for p := v; p.IsValid() && p.CanInterface(); p = p.FieldByName("Context") {
		parent, ok := p.Interface().(context.Context)
		if !ok || parent == nil || fn(parent) {
			break
		}
		if p = reflect.Indirect(p); p.Kind() != reflect.Struct {
			break
		}
	}
}
