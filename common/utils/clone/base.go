package clone

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func ComparablePtr[T comparable](s *T) (d *T) {
	if s == nil {
		return
	}
	d = new(T)
	*d = *s
	return
}

func SliceComparable[T comparable, TS ~[]T](s TS) (d TS) {
	if s == nil {
		return
	}
	d = make(TS, len(s), cap(s))
	copy(d, s)
	return
}

func SliceComparablePtr[T comparable, TS ~[]*T](s TS) (d TS) {
	if s == nil {
		return
	}
	d = make(TS, len(s), cap(s))
	copy(d, s)
	return
}

func TimePtr(s *time.Time) (d *time.Time) {
	if s == nil {
		return
	}
	d = new(time.Time)
	*d = time.Date(s.Year(), s.Month(), s.Day(), s.Hour(), s.Minute(), s.Second(), s.Nanosecond(),
		TimeLocationPtr(s.Location()))
	return
}

func TimeLocationPtr(s *time.Location) (d *time.Location) {
	if s == nil {
		return
	}
	d, _ = time.LoadLocation(s.String())
	return
}

func DecimalPtr(s *decimal.Decimal) (d *decimal.Decimal) {
	if s == nil {
		return
	}
	d = new(decimal.Decimal)
	*d = s.Copy()
	return
}

func GormModelPtr(s *gorm.Model) (d *gorm.Model) {
	if s == nil {
		return
	}
	return &gorm.Model{
		ID:        s.ID,
		CreatedAt: *TimePtr(&s.CreatedAt),
		UpdatedAt: *TimePtr(&s.UpdatedAt),
		DeletedAt: gorm.DeletedAt{
			Time:  *TimePtr(&s.DeletedAt.Time),
			Valid: s.DeletedAt.Valid,
		},
	}
}

type Clonable[T any] interface {
	Clone() T
}

func Slice[T Clonable[T], TS ~[]T](s TS) (d TS) {
	if s == nil {
		return
	}
	d = make(TS, 0, len(s))
	for _, item := range s {
		d = append(d, item.Clone())
	}
	return
}

func Map[T Clonable[T], K comparable](s map[K]T) (d map[K]T) {
	if s == nil {
		return
	}
	d = make(map[K]T, len(s))
	for k, v := range s {
		d[k] = v.Clone()
	}
	return
}
