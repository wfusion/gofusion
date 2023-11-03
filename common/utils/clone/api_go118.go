//go:build go1.18
// +build go1.18

package clone

import (
	"reflect"
	"unsafe"
)

// Clone recursively deep clone v to a new value in heap.
// It assumes that there is no pointer cycle in v,
// e.g. v has a pointer points to v itself.
// If there is a pointer cycle, use Slowly instead.
//
// Clone allocates memory and deeply copies values inside v in depth-first sequence.
// There are a few special rules for following types.
//
//   - Scalar types: all number-like types are copied by value.
//   - func: Copied by value as func is an opaque pointer at runtime.
//   - string: Copied by value as string is immutable by design.
//   - unsafe.Pointer: Copied by value as we don't know what's in it.
//   - chan: A new empty chan is created as we cannot read data inside the old chan.
//
// Unlike many other packages, Clone is able to clone unexported fields of any struct.
// Use this feature wisely.
func Clone[T any](t T) T {
	return cloner.Clone(t).(T)
}

// Slowly recursively deep clone v to a new value in heap.
// It marks all cloned values internally, thus it can clone v with cycle pointer.
//
// Slowly works exactly the same as Clone. See Clone doc for more details.
func Slowly[T any](t T) T {
	return cloner.CloneSlowly(t).(T)
}

// Wrap creates a wrapper of v, which must be a pointer.
// If v is not a pointer, Wrap simply returns v and do nothing.
//
// The wrapper is a deep clone of v's value. It holds a shadow copy to v internally.
//
//	t := &T{Foo: 123}
//	v := Wrap(t).(*T)               // v is a clone of t.
//	reflect.DeepEqual(t, v) == true // v equals t.
//	v.Foo = 456                     // v.Foo is changed, but t.Foo doesn't change.
//	orig := Unwrap(v)               // Use `Unwrap` to discard wrapper and return original value, which is t.
//	orig.(*T) == t                  // orig and t is exactly the same.
//	Undo(v)                         // Use `Undo` to discard any change on v.
//	v.Foo == t.Foo                  // Now, the value of v and t are the same again.
func Wrap[T any](t T) T {
	return wrap(t).(T)
}

// Unwrap returns v's original value if v is a wrapped value.
// Otherwise, simply returns v itself.
func Unwrap[T any](t T) T {
	return unwrap(t).(T)
}

// Undo discards any change made in wrapped value.
// If v is not a wrapped value, nothing happens.
func Undo[T any](t T) {
	undo(t)
}

// MarkAsOpaquePointer marks t as an opaque pointer in heap allocator,
// so that all clone methods will copy t by value.
// If t is not a pointer, MarkAsOpaquePointer ignores t.
//
// Here is a list of types marked as opaque pointers by default:
//   - `elliptic.Curve`, which is `*elliptic.CurveParam` or `elliptic.p256Curve`;
//   - `reflect.Type`, which is `*reflect.rtype` defined in `runtime`.
func MarkAsOpaquePointer(t reflect.Type) {
	markAsOpaquePointer(t)
}

// MarkAsScalar marks t as a scalar type in heap allocator,
// so that all clone methods will copy t by value.
// If t is not struct or pointer to struct, MarkAsScalar ignores t.
//
// In the most cases, it's not necessary to call it explicitly.
// If a struct type contains scalar type fields only, the struct will be marked as scalar automatically.
//
// Here is a list of types marked as scalar by default:
//   - time.Time
//   - reflect.Value
func MarkAsScalar(t reflect.Type) {
	markAsScalar(t)
}

// SetCustomFunc sets a custom clone function for type t in heap allocator.
// If t is not struct or pointer to struct, SetCustomFunc ignores t.
//
// If fn is nil, remove the custom clone function for type t.
func SetCustomFunc(t reflect.Type, fn Func) {
	setCustomFunc(t, fn)
}

// FromHeap creates an allocator which allocate memory from heap.
func FromHeap() *Allocator {
	return fromHeap()
}

// NewAllocator creates an allocator which allocate memory from the pool.
// Both pool and methods are optional.
//
// If methods.New is not nil, the allocator itself is created by calling methods.New.
//
// The pool is a pointer to the memory pool which is opaque to the allocator.
// It's methods' responsibility to allocate memory from the pool properly.
func NewAllocator(pool unsafe.Pointer, methods *AllocatorMethods) (allocator *Allocator) {
	return newAllocator(pool, methods)
}
