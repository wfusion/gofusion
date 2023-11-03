package cases

import (
	"context"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/log"
)

type Eat[T any] interface {
	Egg(T)
}

type Drink interface {
	Water()
}

type PersonEat[T any] struct {
}

func NewPersonEat[T any]() Eat[T] {
	return new(PersonEat[T])
}

func (p *PersonEat[T]) Egg(something T) {
	log.Info(context.TODO(), "鸡蛋好好吃 %+v", something)
}

type PersonDrink struct {
	eating Eat[int]
}

func NewPersonDrink(eating Eat[int]) Drink {
	return &PersonDrink{eating: eating}
}
func (p *PersonDrink) Water() {
	p.eating.Egg(3)
	log.Info(context.TODO(), "要多喝水")
}

type Person struct {
	di.In

	Eating   Eat[int]
	Drinking Drink
}

func (p *Person) Show() {
	p.Eating.Egg(2)
	p.Drinking.Water()
}

type Person2 struct {
	di.In

	Eating   Eat[string]
	Drinking Drink
}

func (p *Person2) Show() {
	p.Eating.Egg("string")
	p.Drinking.Water()
}

type Person3 struct {
	di.In

	Eating   Eat[string]
	Drinking Drink `name:"ddd"`
}

func (p *Person3) Show() {
	p.Eating.Egg("string")
	p.Drinking.Water()
}
