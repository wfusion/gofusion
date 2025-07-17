package config

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
)

// common component names
const (
	ComponentApp           = "App"
	ComponentDebug         = "Debug"
	ComponentRemoteConfig  = "RemoteConfig"
	ComponentCrypto        = "Crypto"
	ComponentMetrics       = "Metrics"
	ComponentTrace         = "Trace"
	ComponentLog           = "Log"
	ComponentDB            = "DB"
	ComponentRedis         = "Redis"
	ComponentKV            = "KV"
	ComponentMongo         = "Mongo"
	ComponentI18n          = "I18n"
	ComponentLock          = "Lock"
	ComponentMessageQueue  = "MQ"
	ComponentHttp          = "Http"
	ComponentCache         = "Cache"
	ComponentCron          = "Cron"
	ComponentAsync         = "Async"
	ComponentGoroutinePool = "GoroutinePool"
)

const (
	DefaultInstanceKey = "default"
)

var (
	// componentOrder common component setup order
	componentOrder = []string{
		ComponentApp,
		ComponentDebug,
		ComponentRemoteConfig,
		ComponentCrypto,
		ComponentLog,
		ComponentMetrics,
		ComponentTrace,
		ComponentRedis,
		ComponentKV,
		ComponentCache,
		ComponentDB,
		ComponentMongo,
		ComponentI18n,
		ComponentLock,
		ComponentMessageQueue,
		ComponentAsync,
		ComponentGoroutinePool,
		ComponentCron,
		ComponentHttp,
	}

	componentLocker sync.RWMutex
	components      []*componentItem
)

func indexComponent(name string) (idx int) {
	for idx = 0; idx < len(componentOrder); idx++ {
		if componentOrder[idx] == name {
			return
		}
	}
	idx = -1
	return
}

type Component struct {
	name                 string
	tag                  string
	constructor          reflect.Value
	constructorInputType reflect.Type
	isCore               bool
	flagString           *string
}

func (c *Component) Clone() (r *Component) {
	return &Component{
		name:                 c.name,
		tag:                  c.tag,
		constructor:          c.constructor,
		constructorInputType: c.constructorInputType,
		isCore:               c.isCore,
		flagString:           c.flagString,
	}
}

type options struct {
	tagList         []string
	isCoreComponent bool
	flagValue       *string
}

type ComponentOption func(*options)

func newOptions() *options {
	return &options{}
}

// WithTag set component struct tags
func WithTag(name, val string) ComponentOption {
	return func(opt *options) {
		opt.tagList = append(opt.tagList, fmt.Sprintf(`%s:"%s"`, name, val))
	}
}

// WithCore mark component as core component, they must be init first
func WithCore() ComponentOption {
	return func(opt *options) {
		opt.isCoreComponent = true
	}
}

func WithFlag(flagValue *string) ComponentOption {
	return func(o *options) {
		o.flagValue = flagValue
	}
}

type componentItem struct {
	name        string
	constructor any
	opt         []ComponentOption
}

func AddComponent(name string, constructor any, opts ...ComponentOption) {
	componentLocker.Lock()
	defer componentLocker.Unlock()
	parseConstructor(constructor)
	components = append(components, &componentItem{name, constructor, opts})
}

func getComponents() []*componentItem {
	componentLocker.RLock()
	defer componentLocker.RUnlock()
	return clone.Clone(components)
}

func parseConstructor(fn any) (fnVal reflect.Value, input reflect.Type) {
	fnVal = reflect.ValueOf(fn)
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		panic(errors.New("component constructor should be a function"))
	}

	// check output
	if typ.NumOut() != 1 {
		panic(errors.New("component constructor should return one finalizer function"))
	}
	retTyp := typ.Out(0)
	if retTyp.Kind() != reflect.Func {
		panic(errors.New("component constructor should return one finalizer function"))
	}
	if retTyp.NumIn() != 0 {
		panic(errors.New("component constructor should return one finalizer function looks like func()"))
	}

	// check input
	fnType := fnVal.Type()
	if n := fnType.NumIn(); n != 1 && (!fnType.IsVariadic() && n != 3) {
		panic(errors.New("component constructor should receive input looks like " +
			"func(context.Context), func(context.Context, *serializableConf, ...utils.OptionExtender)"))
	}
	if fnType.In(0) != constant.ContextType {
		panic(errors.New("component constructor should receive context.Context as first input " +
			"looks like func(context.Context), func(context.Context, *serializableConf, ...utils.OptionExtender)"))
	}

	// wrapper
	switch typ.NumIn() {
	case 1:
		input = reflect.TypeOf(int(0))
		fnVal = reflect.ValueOf(func(ctx context.Context, mock int, _ ...utils.OptionExtender) func() {
			out := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(ctx)})
			if retfn := out[0]; retfn.IsNil() {
				return nil
			} else if obj := retfn.Interface(); obj == nil {
				return nil
			} else if fn, ok := obj.(func()); !ok {
				return nil
			} else {
				return fn
			}
		})
	case 3:
		input = typ.In(1)
		argsType := typ.In(2)
		if argsType.Kind() != reflect.Slice ||
			argsType.Elem() != reflect.TypeOf((*utils.OptionExtender)(nil)).Elem() {
			panic(errors.New("component constructor only receive utils.OptionExtender variadic input"))
		}
	default:
		panic(errors.New("component constructor should receive one or three inputs looks like " +
			"func(context.Context), func(context.Context, *serializableConf, ...utils.OptionExtender)"))
	}
	return
}
