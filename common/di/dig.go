package di

import (
	"fmt"
	"reflect"

	"go.uber.org/dig"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
)

var (
	Dig = NewDI()

	inType     = reflect.TypeOf(In{})
	outType    = reflect.TypeOf(Out{})
	digInType  = reflect.TypeOf(dig.In{})
	digOutType = reflect.TypeOf(dig.Out{})

	Type = reflect.TypeOf(NewDI())
)

type _dig struct {
	*dig.Container
	fields []reflect.StructField
}

func NewDI() DI {
	return &_dig{Container: dig.New()}
}

type provideOption struct {
	name  string
	group string
}

func Name(name string) utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.name = name
	}
}

func Group(group string) utils.OptionFunc[provideOption] {
	return func(p *provideOption) {
		p.group = group
	}
}

func (d *_dig) Invoke(fn any) error { return d.Container.Invoke(fn) }
func (d *_dig) MustInvoke(fn any)   { utils.MustSuccess(d.Container.Invoke(fn)) }
func (d *_dig) Provide(ctor any, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[provideOption](opts...)
	digOpts := make([]dig.ProvideOption, 0, 2)
	if opt.name != "" {
		digOpts = append(digOpts, dig.Name(opt.name))
	}
	if opt.group != "" {
		digOpts = append(digOpts, dig.Group(opt.group))
	}

	defer d.addFields(ctor, opt)
	return d.Container.Provide(ctor, digOpts...)
}
func (d *_dig) MustProvide(ctor any, opts ...utils.OptionExtender) DI {
	utils.MustSuccess(d.Provide(ctor, opts...))
	return d
}
func (d *_dig) Decorate(decorator any) error { return d.Container.Decorate(decorator) }
func (d *_dig) MustDecorate(decorator any) DI {
	utils.MustSuccess(d.Container.Decorate(decorator))
	return d
}

func (d *_dig) String() string {
	return d.Container.String()
}

func (d *_dig) Clear() {
	d.Container = dig.New()
}

// Preload prevent invoke concurrently because invoke is not concurrent safe
// base on: https://github.com/uber-go/dig/issues/241
func (d *_dig) Preload() {
	fields := make([]reflect.StructField, 0, 1+len(d.fields))
	fields = append(fields, reflect.StructField{
		Name:      "In",
		PkgPath:   "",
		Type:      digInType,
		Tag:       "",
		Offset:    0,
		Index:     nil,
		Anonymous: true,
	})
	for i := 0; i < len(d.fields); i++ {
		fields = append(fields, reflect.StructField{
			Name:      fmt.Sprintf("Arg%X", i+1),
			PkgPath:   "",
			Type:      d.fields[i].Type,
			Tag:       d.fields[i].Tag,
			Offset:    0,
			Index:     nil,
			Anonymous: false,
		})
	}
	structType := reflect.StructOf(fields)

	// FIXME: we cannot declare function param type dynamic now
	scope := inspect.GetField[*dig.Scope](d.Container, "scope")
	containerStoreType := inspect.TypeOf("go.uber.org/dig.containerStore")
	containerStoreVal := reflect.ValueOf(scope).Convert(containerStoreType)

	fakeParam := utils.Must(newParam(structType, scope))
	paramType := inspect.TypeOf("go.uber.org/dig.param")
	paramVal := reflect.ValueOf(fakeParam).Convert(paramType)
	buildFn := paramVal.MethodByName("Build")
	returnValList := buildFn.Call([]reflect.Value{containerStoreVal})
	if errVal := returnValList[len(returnValList)-1].Interface(); errVal != nil {
		if err, ok := errVal.(error); ok && err != nil {
			panic(err)
		}
	}
}

func (d *_dig) addFields(ctor any, opt *provideOption) {
	typ := reflect.TypeOf(ctor)
	numOfOut := typ.NumOut()
	for i := 0; i < numOfOut; i++ {
		out := typ.Out(i)
		// ignore error and non-interface nor non-struct out param
		if out == constant.ErrorType ||
			(out.Kind() != reflect.Interface &&
				(out.Kind() != reflect.Struct && !(out.Kind() == reflect.Ptr && out.Elem().Kind() == reflect.Struct))) {
			continue
		}
		if out.Kind() == reflect.Ptr {
			out = out.Elem()
		}

		if !utils.EmbedsType(out, digOutType) {
			var tag reflect.StructTag
			switch {
			case opt.name != "":
				tag = reflect.StructTag(fmt.Sprintf(`name:"%s"`, opt.name))
			case opt.group != "":
				tag = reflect.StructTag(fmt.Sprintf(`group:"%s"`, opt.group))
				out = reflect.SliceOf(out)
			}

			d.fields = append(d.fields, reflect.StructField{Type: out, Tag: tag})
			continue
		}

		// traverse all field
		numOfFields := out.NumField()
		for j := 0; j < numOfFields; j++ {
			f := out.Field(j)

			// ignore dig out
			if f.Type == digOutType || f.Type == outType {
				continue
			}

			d.fields = append(d.fields, f)
		}
	}
}
