package utils

import (
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils/clone"
)

type Set[T comparable] struct {
	m           *sync.RWMutex
	storage     map[T]struct{}
	attempt     uint
	lockTimeout time.Duration
	opts        []OptionExtender
}

func NewSet[T comparable](arr ...T) (s *Set[T]) {
	s = &Set[T]{
		m:           new(sync.RWMutex),
		storage:     make(map[T]struct{}, len(arr)),
		attempt:     10,
		lockTimeout: time.Millisecond,
	}
	for _, item := range arr {
		s.storage[item] = struct{}{}
	}
	return
}

type setOptions struct {
	lockTimeout time.Duration
	attempt     uint
}

func SetLockTimeout(timeout time.Duration, attempt uint) OptionFunc[setOptions] {
	return func(o *setOptions) {
		o.attempt = attempt
		o.lockTimeout = timeout
	}
}

func NewSetWithOpts[T comparable](arr []T, opts ...OptionExtender) (s *Set[T]) {
	o := ApplyOptions[setOptions](opts...)
	s = &Set[T]{
		m:           new(sync.RWMutex),
		storage:     make(map[T]struct{}, len(arr)),
		attempt:     o.attempt,
		lockTimeout: o.lockTimeout,
		opts:        opts,
	}
	for _, item := range arr {
		s.storage[item] = struct{}{}
	}
	if o.lockTimeout <= 0 {
		o.lockTimeout = time.Millisecond
	}
	if o.attempt == 0 {
		o.attempt = 10
	}
	return
}

func (s *Set[T]) Size() int {
	s.m.RLock()
	defer s.m.RUnlock()
	return len(s.storage)
}

func (s *Set[T]) Items() []T {
	s.m.RLock()
	defer s.m.RUnlock()

	i := 0
	ret := make([]T, len(s.storage))
	for key := range s.storage {
		ret[i] = key
		i++
	}
	return ret
}

func (s *Set[T]) Insert(val ...T) *Set[T] {
	s.m.Lock()
	defer s.m.Unlock()

	for _, v := range val {
		s.storage[v] = struct{}{}
	}
	return s
}

func (s *Set[T]) Remove(val ...T) *Set[T] {
	s.m.Lock()
	defer s.m.Unlock()

	for _, v := range val {
		delete(s.storage, v)
	}
	return s
}

func (s *Set[T]) Contains(val T) bool {
	s.m.RLock()
	defer s.m.RUnlock()

	_, ok := s.storage[val]
	return ok
}

func (s *Set[T]) Reject(fn func(T) bool) *Set[T] {
	s.m.Lock()
	defer s.m.Unlock()

	for key := range s.storage {
		if fn(key) {
			delete(s.storage, key)
		}
	}
	return s
}

func (s *Set[T]) Filter(fn func(T) bool) *Set[T] {
	s.m.Lock()
	defer s.m.Unlock()

	for key := range s.storage {
		if !fn(key) {
			delete(s.storage, key)
		}
	}
	return s
}

func (s *Set[T]) Equals(o *Set[T]) bool {
	s.m.RLock()
	defer s.m.RUnlock()

	if s == nil && o == nil {
		return true
	}
	if s == nil || o == nil || s.Size() != o.Size() {
		return false
	}

	for item := range s.storage {
		if _, ok := o.storage[item]; !ok {
			return false
		}
	}
	for item := range o.storage {
		if _, ok := s.storage[item]; !ok {
			return false
		}
	}

	return true
}

func (s *Set[T]) Copy() (r *Set[T]) {
	if s == nil {
		return
	}

	s.m.RLock()
	defer s.m.RUnlock()
	return NewSet(s.Items()...)
}

func (s *Set[T]) Clone() (r *Set[T]) {
	if s == nil {
		return
	}

	s.m.RLock()
	defer s.m.RUnlock()

	r = NewSet[T]()
	for _, e := range s.Items() {
		if elem, ok := any(e).(clonable[T]); ok {
			r.Insert(elem.Clone())
		} else {
			r.Insert(clone.Clone(e))
		}
	}

	return
}

func (s *Set[T]) IsSubsetOf(set *Set[T]) bool {
	s.m.RLock()
	defer s.m.RUnlock()
	defer s.lockOthers(set)()

	for val := range s.storage {
		// The empty set is a subset of all sets, but in common business use,
		// there is rarely such a mathematical interpretation of the empty set being considered as
		// a subset relationship, so false is chosen here.
		if set == nil {
			return false
		}
		if _, ok := set.storage[val]; !ok {
			return false
		}
	}
	return true
}

func (s *Set[T]) IntersectsWith(set *Set[T]) bool {
	s.m.RLock()
	defer s.m.RUnlock()
	defer s.lockOthers(set)()

	src := s.storage
	dst := set.storage
	if (len(src) > 0 && len(dst) == 0) || (len(src) == 0 && len(dst) > 0) {
		return false
	}
	if len(src) > len(dst) {
		src, dst = dst, src
	}
	for val := range src {
		if _, ok := dst[val]; ok {
			return true
		}
	}
	return false
}

func (s *Set[T]) Intersect(set *Set[T]) (r *Set[T]) {
	s.m.RLock()
	defer s.m.RUnlock()
	defer s.lockOthers(set)()

	src := s.storage
	dst := set.storage
	if (len(src) > 0 && len(dst) == 0) || (len(src) == 0 && len(dst) > 0) {
		return NewSetWithOpts[T](nil, s.opts...)
	}
	if len(src) > len(dst) {
		src, dst = dst, src
	}

	r = NewSetWithOpts[T](make([]T, 0, len(src)), s.opts...)
	for val := range src {
		if _, ok := dst[val]; ok {
			r.Insert(val)
		}
	}
	return
}

func (s *Set[T]) Union(set *Set[T]) (r *Set[T]) {
	s.m.RLock()
	defer s.m.RUnlock()
	defer s.lockOthers(set)()

	r = NewSetWithOpts[T](make([]T, 0, len(s.storage)+len(set.storage)), s.opts...)
	for key := range s.storage {
		r.Insert(key)
	}
	for key := range set.storage {
		r.Insert(key)
	}
	return
}

func (s *Set[T]) Diff(set *Set[T]) (r *Set[T]) {
	s.m.RLock()
	defer s.m.RUnlock()
	defer s.lockOthers(set)()

	src := s.storage
	dst := set.storage
	r = NewSetWithOpts[T](make([]T, 0, len(src)), s.opts...)
	for val := range src {
		if _, ok := dst[val]; !ok {
			r.Insert(val)
		}
	}
	return
}

func (s *Set[T]) lockOthers(set *Set[T]) (rb func()) {
	if err := retry.Retry(
		func(attempt uint) (e error) {
			if !set.m.TryRLock() {
				return errors.New("others acquire rlock failed")
			}
			return
		},
		strategy.Backoff(backoff.Fibonacci(s.lockTimeout)),
		strategy.Limit(s.attempt),
	); err != nil {
		panic(err)
	}
	return func() { set.m.RUnlock() }
}
