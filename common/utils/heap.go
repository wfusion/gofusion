package utils

import (
	"container/heap"
	"errors"
)

var (
	ErrOutOfRange = errors.New("out of range")
)

type _heap[E any] struct {
	d   []E
	cmp func(e1, e2 E) bool
}

func (h *_heap[E]) Len() int           { return len(h.d) }
func (h *_heap[E]) Less(i, j int) bool { return h.cmp(h.d[i], h.d[j]) }
func (h *_heap[E]) Swap(i, j int)      { h.d[i], h.d[j] = h.d[j], h.d[i] }
func (h *_heap[E]) Push(x any)         { v := append(h.d, x.(E)); h.d = v }
func (h *_heap[E]) Pop() (x any)       { x, h.d = h.d[len(h.d)-1], h.d[0:len(h.d)-1]; return x }

// Heap base on generics to build a heap tree for any type
type Heap[E any] struct {
	data *_heap[E]
}

// Push pushes the element x onto the heap.
// The complexity is O(log n) where n = h.Len().
func (h *Heap[E]) Push(v E) { heap.Push(h.data, v) }

// Pop removes and returns the minimum element (according to Less) from the heap.
// The complexity is O(log n) where n = h.Len().
// Pop is equivalent to Remove(h, 0).
func (h *Heap[E]) Pop() E { return heap.Pop(h.data).(E) }

func (h *Heap[E]) Element(index int) (e E, err error) {
	if index < 0 || index >= h.data.Len() {
		return e, ErrOutOfRange
	}
	return h.data.d[index], nil
}

// Remove removes and returns the element at index i from the heap.
// The complexity is O(log n) where n = h.Len().
func (h *Heap[E]) Remove(index int) E { return heap.Remove(h.data, index).(E) }
func (h *Heap[E]) Len() int           { return len(h.data.d) }

// Copy heap
func (h *Heap[E]) Copy() *Heap[E] {
	ret := &_heap[E]{cmp: h.data.cmp}
	ret.d = make([]E, len(h.data.d))
	copy(ret.d, h.data.d)
	heap.Init(ret)
	return &Heap[E]{data: ret}
}

// NewHeap return an initial Heap pointer
func NewHeap[E any](t []E, cmp func(e1, e2 E) bool) *Heap[E] {
	ret := &_heap[E]{d: t, cmp: cmp}
	heap.Init(ret)
	return &Heap[E]{data: ret}
}
