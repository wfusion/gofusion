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
	inType     = reflect.TypeOf(In{})
	outType    = reflect.TypeOf(Out{})
	digInType  = reflect.TypeOf(dig.In{})
	digOutType = reflect.TypeOf(dig.Out{})

	Type = reflect.TypeOf(NewDI())
)

type _dig struct {
	scope  *dig.Scope
	fields []reflect.StructField
}

func (d *_dig) Invoke(fn any) error { return d.scope.Invoke(fn) }
func (d *_dig) MustInvoke(fn any)   { utils.MustSuccess(d.scope.Invoke(fn)) }
func (d *_dig) Provide(ctor any, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[provideOption](opts...)
	digOpts := make([]dig.ProvideOption, 0, 2)
	if opt.name != "" {
		digOpts = append(digOpts, dig.Name(opt.name))
	}
	if opt.group != "" {
		digOpts = append(digOpts, dig.Group(opt.group))
	}
	if opt.export {
		digOpts = append(digOpts, dig.Export(true))
	}

	defer d.addFields(ctor, opt)
	return d.scope.Provide(ctor, digOpts...)
}
func (d *_dig) MustProvide(ctor any, opts ...utils.OptionExtender) DI {
	utils.MustSuccess(d.Provide(ctor, opts...))
	return d
}
func (d *_dig) Decorate(decorator any) error { return d.scope.Decorate(decorator) }
func (d *_dig) MustDecorate(decorator any) DI {
	utils.MustSuccess(d.scope.Decorate(decorator))
	return d
}
func (d *_dig) Populate(objs ...any) error  { return d.populate(objs...) }
func (d *_dig) MustPopulate(objs ...any) DI { utils.MustSuccess(d.Populate(objs...)); return d }

func (d *_dig) Scope(name string, opts ...utils.OptionExtender) DI {
	return &_dig{
		scope:  d.scope.Scope(name),
		fields: d.fields,
	}
}

func (d *_dig) String() string {
	return d.scope.String()
}

func (d *_dig) Clear() {
	d.scope = inspect.GetField[*dig.Scope](dig.New(), "scope")
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
	containerStoreType := inspect.TypeOf("go.uber.org/dig.containerStore")
	containerStoreVal := reflect.ValueOf(d.scope).Convert(containerStoreType)

	fakeParam := utils.Must(newParam(structType, d.scope))
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

func (d *_dig) populate(targets ...any) error {
	// Validate all targets are non-nil pointers.
	fields := make([]reflect.StructField, len(targets)+1)
	fields[0] = reflect.StructField{
		Name:      "In",
		Type:      reflect.TypeOf(In{}),
		Anonymous: true,
	}
	for i, t := range targets {
		if t == nil {
			return fmt.Errorf("failed to Populate: target %v is nil", i+1)
		}
		var (
			rt  = reflect.TypeOf(t)
			tag reflect.StructTag
		)
		if rt.Kind() != reflect.Ptr {
			return fmt.Errorf("failed to Populate: target %v is not a pointer type, got %T", i+1, t)
		}
		fields[i+1] = reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: rt.Elem(),
			Tag:  tag,
		}
	}

	// Build a function that looks like:
	//
	// func(t1 T1, t2 T2, ...) {
	//   *targets[0] = t1
	//   *targets[1] = t2
	//   [...]
	// }
	//
	fnType := reflect.FuncOf([]reflect.Type{reflect.StructOf(fields)}, nil, false /* variadic */)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		arg := args[0]
		for i, target := range targets {
			reflect.ValueOf(target).Elem().Set(arg.Field(i + 1))
		}
		return nil
	})
	return d.Invoke(fn.Interface())
}
