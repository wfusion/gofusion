package utils

import "sort"

func Sort[E any](data []E, cmp func(e1, e2 E) int) {
	sortObj := sortable[E]{data: data, cmp: cmp}
	sort.Sort(sortObj)
}

func SortStable[E any](data []E, cmp func(e1, e2 E) int) {
	sortObj := sortable[E]{data: data, cmp: cmp}
	sort.Stable(sortObj)
}

type sortable[E any] struct {
	data []E
	cmp  func(e1, e2 E) int
}

func (s sortable[E]) Len() int           { return len(s.data) }
func (s sortable[E]) Swap(i, j int)      { s.data[i], s.data[j] = s.data[j], s.data[i] }
func (s sortable[E]) Less(i, j int) bool { return s.cmp(s.data[i], s.data[j]) < 0 }
