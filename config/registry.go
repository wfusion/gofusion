package config

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/utils/serialize/json"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
)

var (
	Registry = &registry{di: di.Dig, app: di.Fx, initOnce: new(sync.Once), closeCh: make(chan struct{})}

	initLocker   sync.RWMutex
	registryLock sync.RWMutex
	registryMap  = map[string]Configurable{"": Registry}
)

const (
	componentConfigFieldName = "Base"
)

func Use(appName string, opts ...utils.OptionExtender) Configurable {
	registryLock.RLock()
	defer registryLock.RUnlock()
	cfg, ok := registryMap[appName]
	if !ok {
		panic(errors.Errorf("app register config not found: %s", appName))
	}
	return cfg
}

func New(appName string) Configurable {
	registryLock.Lock()
	defer registryLock.Unlock()
	if reg, ok := registryMap[appName]; ok {
		return reg
	}

	reg := &registry{
		di:       di.NewDI(),
		app:      di.New(),
		appName:  appName,
		initOnce: new(sync.Once),
		closeCh:  make(chan struct{}),
	}
	registryMap[appName] = reg
	return reg
}

type registry struct {
	di                 di.DI
	app                di.App
	appName            string
	debug              bool
	loadComponentsOnce sync.Once
	initOnce           *sync.Once
	initWg             sync.WaitGroup
	closeCh            chan struct{}

	componentList      []*Component
	businessConfig     any
	businessConfigType reflect.Type
	componentConfigs   any
}

type initOption struct {
	debug          bool
	bizCtx         context.Context
	customLoadFunc loadConfigFunc
	filenames      []string
}

func Ctx(ctx context.Context) utils.OptionFunc[initOption] {
	return func(o *initOption) {
		o.bizCtx = ctx
	}
}

func Loader(fn func(any, ...utils.OptionExtender)) utils.OptionFunc[initOption] {
	return func(o *initOption) {
		o.customLoadFunc = fn
	}
}

func Files(filenames []string) utils.OptionFunc[initOption] {
	return func(o *initOption) {
		o.filenames = filenames
	}
}

func Debug() utils.OptionFunc[initOption] {
	return func(o *initOption) {
		o.debug = true
	}
}

func (r *registry) Init(businessConfig any, opts ...utils.OptionExtender) (gracefully func()) {
	initLocker.Lock()
	defer initLocker.Unlock()

	r.initWg.Add(1)
	r.initOnce.Do(func() {
		opt := utils.ApplyOptions[initOption](opts...)
		r.debug = opt.debug
		r.closeCh = make(chan struct{})

		// context
		parent := context.Background()
		if opt.bizCtx != nil {
			parent = opt.bizCtx
		}

		// load config function
		loadFn := loadConfig
		if opt.customLoadFunc != nil {
			loadFn = opt.customLoadFunc
		}

		gracefully = r.initByConfigFile(parent, businessConfig, loadFn, opts...)
	})
	if gracefully == nil {
		// give back
		reflect.Indirect(reflect.ValueOf(businessConfig)).Set(reflect.ValueOf(r.businessConfig))

		once := new(sync.Once)
		gracefully = func() {
			once.Do(func() {
				r.initWg.Done()
			})
		}
	}
	return
}

func (r *registry) AddComponent(name string, constructor any, opts ...ComponentOption) {
	if name[0] < 'A' || name[0] > 'Z' {
		panic("component name should start with A-Z")
	}
	for idx, com := range r.componentList {
		if com.name == name {
			r.componentList = append(r.componentList[:idx], r.componentList[idx+1:]...)
		}
	}
	opt := newOptions()
	for _, fn := range opts {
		fn(opt)
	}

	com := &Component{
		name:       name,
		isCore:     opt.isCoreComponent,
		flagString: opt.flagValue,
	}

	hasYamlTag := false
	hasJsonTag := false
	hasTomlTag := false
	for _, tag := range opt.tagList {
		hasYamlTag = strings.HasPrefix(tag, "`yaml:")
		hasJsonTag = strings.HasPrefix(tag, "`json:")
		hasTomlTag = strings.HasPrefix(tag, "`toml:")
	}
	lowerName := strcase.ToSnake(name)
	if name == ComponentI18n {
		lowerName = strings.ToLower(name)
	}
	if !hasYamlTag {
		opt.tagList = append(opt.tagList, fmt.Sprintf(`yaml:"%s"`, lowerName))
	}
	if !hasJsonTag {
		opt.tagList = append(opt.tagList, fmt.Sprintf(`json:"%s"`, lowerName))
	}
	if !hasTomlTag {
		opt.tagList = append(opt.tagList, fmt.Sprintf(`toml:"%s"`, lowerName))
	}
	if len(opt.tagList) > 0 {
		com.tag = strings.Join(opt.tagList, " ")
	}

	com.constructor, com.constructorInputType = parseConstructor(constructor)

	r.addComponent(com)
}

func (r *registry) LoadComponentConfig(name string, componentConfig any) (err error) {
	val := reflect.ValueOf(componentConfig)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr {
		return errors.New("componentConfig should be pointer")
	}

	var found bool
	for _, com := range r.componentList {
		if com.name == name {
			found = true
			break
		}
	}
	if !found {
		return errors.Errorf("no such component [%s]", name)
	}

	// load config
	if r.componentConfigs == nil {
		return
	}
	componentConfigsValue := utils.IndirectValue(reflect.ValueOf(clone.Clone(r.componentConfigs)))
	if !componentConfigsValue.IsValid() {
		return errors.Errorf("component configs not initialize now [%s]", name)
	}
	componentConfigValue := componentConfigsValue.FieldByName(componentConfigFieldName).FieldByName(name)

	if componentConfigValue.Type().Kind() == reflect.Ptr {
		if componentConfigValue.IsNil() {
			return
		}
		componentConfigValue = componentConfigValue.Elem()
	}
	if componentConfigValue.Type() == typ.Elem() || componentConfigValue.Type().ConvertibleTo(typ.Elem()) {
		val.Elem().Set(reflect.ValueOf(clone.Clone(componentConfigValue.Convert(typ.Elem()).Interface())))
		return
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           componentConfig,
		TagName:          "yaml",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return
	}
	return decoder.Decode(componentConfigValue.Interface())
}

func (r *registry) GetAllConfigs() any {
	val := reflect.New(r.makeAllConfigStruct())
	derefVal := reflect.Indirect(val)

	// business configs
	businessConfigsVal := reflect.Indirect(reflect.ValueOf(r.businessConfig))
	numFields := businessConfigsVal.NumField()
	for i := 0; i < numFields; i++ {
		derefVal.Field(i + 1).Set(businessConfigsVal.Field(i))
	}

	// component configs
	derefComponentConfigsVal := derefVal.FieldByName(componentConfigFieldName)
	componentConfigsVal := reflect.Indirect(reflect.ValueOf(r.componentConfigs)).FieldByName(componentConfigFieldName)
	numFields = componentConfigsVal.NumField()
	for i := 0; i < numFields; i++ {
		derefComponentConfigsVal.Field(i).Set(componentConfigsVal.Field(i))
	}
	return clone.Clone(val.Interface())
}

func (r *registry) initByConfigFile(parent context.Context, businessConfig any,
	loadFn loadConfigFunc, opts ...utils.OptionExtender) func() {
	r.loadComponents()
	r.checkBusinessConfig(businessConfig)

	businessConfigVal := reflect.ValueOf(businessConfig)
	r.businessConfigType = utils.IndirectType(businessConfigVal.Type())
	r.businessConfig = reflect.New(r.businessConfigType).Interface()
	r.componentConfigs = reflect.New(r.makeComponentsConfigStruct()).Interface()

	r.initAllConfigByLoadFunc(loadFn, opts...)
	r.initAllConfigByFlag()

	appName := r.AppName()
	registryLock.Lock()
	if _, ok := registryMap[appName]; !ok {
		registryMap[appName] = r
	}
	registryLock.Unlock()

	// decrypt
	CryptoDecryptByTag(r.businessConfig, AppName(r.AppName()))
	CryptoDecryptByTag(r.componentConfigs, AppName(r.AppName()))

	// give back
	reflect.Indirect(reflect.ValueOf(businessConfig)).Set(reflect.ValueOf(r.businessConfig))

	return r.initComponents(parent)
}

func (r *registry) getBaseObject() reflect.Value {
	return reflect.Indirect(reflect.ValueOf(r.componentConfigs)).FieldByName(componentConfigFieldName)
}

func (r *registry) makeComponentsConfigStruct() reflect.Type {
	fieldList := r.makeComponentsConfigFields()
	return reflect.StructOf([]reflect.StructField{
		{
			Name:      componentConfigFieldName,
			Type:      reflect.StructOf(fieldList),
			Tag:       `yaml:"base" json:"base" toml:"base"`,
			Anonymous: true,
		},
	})
}

func (r *registry) makeComponentsConfigFields() []reflect.StructField {
	fieldList := make([]reflect.StructField, len(r.componentList))
	for i := 0; i < len(r.componentList); i++ {
		component := r.componentList[i]
		fieldList[i] = reflect.StructField{
			Name: component.name,
			Type: component.constructorInputType,
			Tag:  reflect.StructTag(component.tag),
		}
	}

	return fieldList
}

func (r *registry) makeAllConfigStruct() reflect.Type {
	/* AllConfig struct may look like below
	type AllConfig struct {
		XXXBase struct {
			Debug       bool
			App         string
			DB          map[string]*db.Conf
			Redis       map[string]*redis.Conf
			Log         *log.Conf
			...
		} `yaml:"base" json:"base" toml:"base"`

		BusinessConfigField1
	    BusinessConfigField2
		BusinessConfigField3

		...
	}
	*/

	numFields := r.businessConfigType.NumField()
	fieldList := make([]reflect.StructField, 0, numFields+1)
	fieldList = append(fieldList, reflect.StructField{
		Name:      componentConfigFieldName,
		Type:      reflect.StructOf(r.makeComponentsConfigFields()),
		Tag:       `yaml:"base" json:"base" toml:"base"`,
		Anonymous: true,
	})
	for i := 0; i < numFields; i++ {
		fieldList = append(fieldList, r.businessConfigType.Field(i))
	}

	return reflect.StructOf(fieldList)
}

func (r *registry) loadComponents() {
	r.loadComponentsOnce.Do(func() {
		// app
		r.AddComponent(ComponentApp, func(context.Context, string, ...utils.OptionExtender) func() { return nil },
			WithTag("yaml", "app"), WithTag("json", "app"), WithTag("toml", "app"),
			WithFlag(utils.AnyPtr("null")),
		)

		// debug
		r.AddComponent(ComponentDebug, func(context.Context, bool, ...utils.OptionExtender) func() { return nil },
			WithTag("yaml", "debug"), WithTag("json", "debug"), WithTag("toml", "debug"),
			WithFlag(utils.AnyPtr("null")),
		)

		// crypto
		r.AddComponent(ComponentCrypto, CryptoConstruct,
			WithTag("yaml", "crypto"), WithTag("json", "crypto"), WithTag("toml", "crypto"),
			WithFlag(&cryptoFlagString),
		)

		for _, item := range getComponents() {
			r.AddComponent(item.name, item.constructor, item.opt...)
		}

		/* example */
		// registry.AddComponent("ComponentExample", func(context.Context, string) func() { return nil },
		//    WithTag("custom_tag", "val"), WithTag("yaml", "val"))
	})
}

func (r *registry) initAllConfigByLoadFunc(loadFn loadConfigFunc, opts ...utils.OptionExtender) {
	if loadFn != nil {
		loadFn(r.businessConfig, opts...)
		loadFn(r.componentConfigs, opts...)
	}
}

func (r *registry) initAllConfigByFlag() {
	configVal := utils.IndirectValue(reflect.ValueOf(r.componentConfigs)).FieldByName(componentConfigFieldName)
	for i := 0; i < len(r.componentList); i++ {
		com := r.componentList[i]
		if utils.IsStrPtrBlank(com.flagString) {
			continue
		}
		switch com.name {
		case ComponentApp:
			if utils.IsStrNotBlank(appFlagString) {
				configVal.FieldByName(com.name).SetString(appFlagString)
			}
		case ComponentDebug:
			if debugFlag {
				configVal.FieldByName(com.name).SetBool(debugFlag)
			}
		default:
			comValp := configVal.FieldByName(com.name).Addr()
			utils.MustSuccess(json.Unmarshal([]byte(*com.flagString), comValp.Interface()))

			// process defaults
			_ = utils.ParseTag(comValp.Interface(), utils.ParseTagName("default"),
				utils.ParseTagUnmarshalType(utils.UnmarshalTypeYaml))
		}
	}

	if len(appBizFlagString) > 0 {
		utils.MustSuccess(json.Unmarshal([]byte(appBizFlagString), &r.businessConfig))
	}
}

func (r *registry) initComponents(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	ctxVal := reflect.ValueOf(ctx)
	o1 := reflect.ValueOf(utils.OptionExtender(AppName(r.appName)))
	o2 := reflect.ValueOf(utils.OptionExtender(DI(r.di)))
	o3 := reflect.ValueOf(utils.OptionExtender(App(r.app)))

	baseObject := r.getBaseObject()
	destructors := make([]reflect.Value, 0, len(r.componentList))
	componentNames := make([]string, 0, len(r.componentList))
	hasCallbackComponentNames := make([]string, 0, len(r.componentList))
	for i := 0; i < len(r.componentList); i++ {
		com := r.componentList[i]
		comArgs := reflect.ValueOf(clone.Clone(baseObject.FieldByName(com.name).Interface()))
		componentNames = append(componentNames, com.name)
		if out := com.constructor.Call([]reflect.Value{ctxVal, comArgs, o1, o2, o3}); len(out) > 0 && !out[0].IsNil() {
			destructors = append(destructors, out[0])
			hasCallbackComponentNames = append(hasCallbackComponentNames, com.name)
		}
	}

	/* print summary to stdout */
	pid := syscall.Getpid()
	app := r.AppName()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	log.Printf("%v [Gofusion] %s initialized total %d components below: %s\n",
		pid, app, len(componentNames), strings.Join(componentNames, ", "))

	once := new(sync.Once)
	return func() {
		once.Do(func() {
			initLocker.Lock()
			defer initLocker.Unlock()

			defer close(r.closeCh)

			r.initWg.Done()
			r.initWg.Wait()
			cancel()
			for i := len(destructors) - 1; i >= 0; i-- {
				log.Printf("%v [Gofusion] %s %s exiting...", pid, app, hasCallbackComponentNames[i])
				destructors[i].Call(nil)
				log.Printf("%v [Gofusion] %s %s exited", pid, app, hasCallbackComponentNames[i])
			}

			r.di.Clear()
			r.app.Clear()
			r.businessConfig = nil
			r.componentConfigs = nil
			r.initOnce = new(sync.Once)
		})
	}
}

func (r *registry) addComponent(com *Component) {
	firstNonCoreComIndex := -1
	for i, cp := range r.componentList {
		if !cp.isCore {
			firstNonCoreComIndex = i
			break
		}
	}
	if !com.isCore || firstNonCoreComIndex == -1 {
		r.componentList = append(r.componentList, com)
		sort.SliceStable(r.componentList, func(i, j int) bool {
			// core component would not be sorted
			if r.componentList[i].isCore || r.componentList[j].isCore {
				return false
			}

			orderA := indexComponent(r.componentList[i].name)
			if orderA == -1 {
				return false
			}
			orderB := indexComponent(r.componentList[j].name)
			if orderB == -1 {
				return true
			}

			return orderA < orderB
		})

		return
	}
	list := make([]*Component, len(r.componentList)+1)
	for i := range list {
		if i < firstNonCoreComIndex {
			list[i] = r.componentList[i]
		} else if i == firstNonCoreComIndex {
			list[i] = com
		} else {
			list[i] = r.componentList[i-1]
		}
	}

	r.componentList = list
}

func (r *registry) checkBusinessConfig(businessConfig any) {
	typ := reflect.TypeOf(businessConfig)
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Ptr {
		panic(errors.New("businessConfig should be a **struct"))
	}
}
