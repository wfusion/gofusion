package utils

import (
	"sync"
)

type Set[T comparable] struct {
	m       *sync.RWMutex
	storage map[T]struct{}
}

func NewSet[T comparable](arr ...T) (s *Set[T]) {
	s = &Set[T]{
		m:       new(sync.RWMutex),
		storage: make(map[T]struct{}, len(arr)),
	}
	for _, item := range arr {
		s.storage[item] = struct{}{}
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

func (s *Set[T]) IsSubsetOf(set *Set[T]) bool {
	s.m.RLock()
	defer s.m.RUnlock()

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

	src := set.storage
	dst := s.storage
	if len(src) > len(dst) {
		src, dst = dst, src
	}
	for val := range src {
		if dst == nil {
			return false
		}
		if _, ok := dst[val]; ok {
			return true
		}
	}
	return false
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

func (s *Set[T]) Clone() (r *Set[T]) {
	s.m.RLock()
	defer s.m.RUnlock()
	return NewSet(s.Items()...)
}
