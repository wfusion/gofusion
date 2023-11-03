package utils

import (
	"bytes"
	"github.com/spf13/cast"
	"sync"
)

const (
	maxBytesPoolable = 16*1024 + 512 // 16.5kb
)

var (
	// BytesBufferPool 64 is bytes.Buffer smallBufferSize, which is an initial allocation minimal capacity.
	BytesBufferPool = NewPool(
		func() *bytes.Buffer { return bytes.NewBuffer(make([]byte, 0, 64)) },
		PoolableEvictFunc(poolBytesBufferEvict),
	)
	BytesPool = NewPool(
		func() poolBytes { return make([]byte, 0, 64) },
		PoolableEvictFunc(poolBytesEvict),
	)
)

type Poolable[T any] interface {
	Get(initialized any) (T, func())
	Put(obj T)
}

type poolableOption[T any] struct {
	evict func(obj T) bool
}

func PoolableEvictFunc[T any](fn func(obj T) bool) OptionFunc[poolableOption[T]] {
	return func(o *poolableOption[T]) {
		o.evict = fn
	}
}

type poolResettableA[T any] interface{ Reset(obj any) T }
type poolResettableB interface{ Reset() }
type poolResettableC interface{ Reset() error }
type poolResettableD interface{ Reset(obj any) }
type poolResettableE interface{ Reset(obj any) error }
type poolResettableF[T any] interface{ Reset() T }

func NewPool[T any](newFn func() T, opts ...OptionExtender) Poolable[T] {
	opt := ApplyOptions[poolableOption[T]](opts...)
	return &poolSealer[T]{
		option: opt,
		newFn:  newFn,
		inner: &sync.Pool{
			New: func() any {
				return any(newFn())
			},
		},
	}
}

type poolSealer[T any] struct {
	option *poolableOption[T]
	inner  *sync.Pool
	newFn  func() T
}

func (p *poolSealer[T]) Get(initialized any) (T, func()) {
	obj, ok := p.inner.Get().(T)
	if !ok {
		obj = p.newFn()
	}

	switch resettable := any(obj).(type) {
	case poolResettableA[T]:
		obj = resettable.Reset(initialized)
	case poolResettableB:
		resettable.Reset()
	case poolResettableC:
		MustSuccess(resettable.Reset())
	case poolResettableD:
		resettable.Reset(initialized)
	case poolResettableE:
		MustSuccess(resettable.Reset(initialized))
	case poolResettableF[T]:
		obj = resettable.Reset()
	}

	once := new(sync.Once)
	return obj, func() {
		once.Do(func() {
			if p.option.evict == nil || !p.option.evict(obj) {
				p.Put(obj)
			}
		})
	}
}

func (p *poolSealer[T]) Put(obj T) { p.inner.Put(obj) }

type poolBytes []byte

func (p poolBytes) Reset(initLen any) poolBytes {
	iLen := cast.ToInt(initLen)
	pp := p
	if cap(p) < cast.ToInt(iLen) {
		pp = make([]byte, iLen)
	}

	return pp[:iLen]
}

func poolBytesBufferEvict(b *bytes.Buffer) bool {
	return b.Cap() >= maxBytesPoolable
}

func poolBytesEvict(p poolBytes) bool {
	return cap(p) >= maxBytesPoolable
}
