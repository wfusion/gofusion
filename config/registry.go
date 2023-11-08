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

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
)

var (
	Registry = &registry{di: di.Dig, initOnce: new(sync.Once), closeCh: make(chan struct{})}

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
		appName:  appName,
		initOnce: new(sync.Once),
		closeCh:  make(chan struct{}),
	}
	registryMap[appName] = reg
	return reg
}

type registry struct {
	di                 di.DI
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

func (p *registry) Init(businessConfig any, opts ...utils.OptionExtender) (gracefully func()) {
	initLocker.Lock()
	defer initLocker.Unlock()

	p.initWg.Add(1)
	p.initOnce.Do(func() {
		opt := utils.ApplyOptions[initOption](opts...)
		p.debug = opt.debug
		p.closeCh = make(chan struct{})

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

		gracefully = p.initByConfigFile(parent, businessConfig, loadFn, opts...)
	})
	if gracefully == nil {
		// give back
		reflect.Indirect(reflect.ValueOf(businessConfig)).Set(reflect.ValueOf(p.businessConfig))

		once := new(sync.Once)
		gracefully = func() {
			once.Do(func() {
				p.initWg.Done()
			})
		}
	}
	return
}

func (p *registry) AddComponent(name string, constructor any, opts ...ComponentOption) {
	if name[0] < 'A' || name[0] > 'Z' {
		panic("component name should start with A-Z")
	}
	for idx, com := range p.componentList {
		if com.Name == name {
			p.componentList = append(p.componentList[:idx], p.componentList[idx+1:]...)
		}
	}
	opt := newOptions()
	for _, fn := range opts {
		fn(opt)
	}

	com := &Component{
		Name:   name,
		isCore: opt.IsCoreComponent,
	}

	hasYamlTag := false
	hasJsonTag := false
	hasTomlTag := false
	for _, tag := range opt.TagList {
		hasYamlTag = strings.HasPrefix(tag, "`yaml:")
		hasJsonTag = strings.HasPrefix(tag, "`json:")
		hasTomlTag = strings.HasPrefix(tag, "`toml:")
	}
	lowerName := strcase.ToSnake(name)
	if name == ComponentI18n {
		lowerName = strings.ToLower(name)
	}
	if !hasYamlTag {
		opt.TagList = append(opt.TagList, fmt.Sprintf(`yaml:"%s"`, lowerName))
	}
	if !hasJsonTag {
		opt.TagList = append(opt.TagList, fmt.Sprintf(`json:"%s"`, lowerName))
	}
	if !hasTomlTag {
		opt.TagList = append(opt.TagList, fmt.Sprintf(`toml:"%s"`, lowerName))
	}
	if len(opt.TagList) > 0 {
		com.Tag = strings.Join(opt.TagList, " ")
	}

	com.Constructor, com.ConstructorInputType = parseConstructor(constructor)

	p.addComponent(com)
}

func (p *registry) LoadComponentConfig(name string, componentConfig any) (err error) {
	val := reflect.ValueOf(componentConfig)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr {
		return errors.New("componentConfig should be pointer")
	}

	var found bool
	for _, com := range p.componentList {
		if com.Name == name {
			found = true
			break
		}
	}
	if !found {
		return errors.Errorf("no such component [%s]", name)
	}

	// load config
	if p.componentConfigs == nil {
		return
	}
	componentConfigsValue := utils.IndirectValue(reflect.ValueOf(clone.Clone(p.componentConfigs)))
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

func (p *registry) GetAllConfigs() any {
	val := reflect.New(p.makeAllConfigStruct())
	derefVal := reflect.Indirect(val)

	// business configs
	businessConfigsVal := reflect.Indirect(reflect.ValueOf(p.businessConfig))
	numFields := businessConfigsVal.NumField()
	for i := 0; i < numFields; i++ {
		derefVal.Field(i + 1).Set(businessConfigsVal.Field(i))
	}

	// component configs
	derefComponentConfigsVal := derefVal.FieldByName(componentConfigFieldName)
	componentConfigsVal := reflect.Indirect(reflect.ValueOf(p.componentConfigs)).FieldByName(componentConfigFieldName)
	numFields = componentConfigsVal.NumField()
	for i := 0; i < numFields; i++ {
		derefComponentConfigsVal.Field(i).Set(componentConfigsVal.Field(i))
	}
	return clone.Clone(val.Interface())
}

func (p *registry) initByConfigFile(parent context.Context, businessConfig any,
	loadFn loadConfigFunc, opts ...utils.OptionExtender) func() {
	p.loadComponents()
	p.checkBusinessConfig(businessConfig)
	p.initAllConfigByLoadFunc(businessConfig, loadFn, opts...)

	appName := p.AppName()
	registryLock.Lock()
	if _, ok := registryMap[appName]; !ok {
		registryMap[appName] = p
	}
	registryLock.Unlock()

	// decrypt
	CryptoDecryptByTag(p.businessConfig, AppName(p.AppName()))
	CryptoDecryptByTag(p.componentConfigs, AppName(p.AppName()))

	// give back
	reflect.Indirect(reflect.ValueOf(businessConfig)).Set(reflect.ValueOf(p.businessConfig))

	return p.initComponents(parent)
}

func (p *registry) getBaseObject() reflect.Value {
	return reflect.Indirect(reflect.ValueOf(p.componentConfigs)).FieldByName(componentConfigFieldName)
}

func (p *registry) makeComponentsConfigStruct() reflect.Type {
	fieldList := p.makeComponentsConfigFields()
	return reflect.StructOf([]reflect.StructField{
		{
			Name:      componentConfigFieldName,
			Type:      reflect.StructOf(fieldList),
			Tag:       `yaml:"base" json:"base" toml:"base"`,
			Anonymous: true,
		},
	})
}

func (p *registry) makeComponentsConfigFields() []reflect.StructField {
	fieldList := make([]reflect.StructField, len(p.componentList))
	for i := 0; i < len(p.componentList); i++ {
		component := p.componentList[i]
		fieldList[i] = reflect.StructField{
			Name: component.Name,
			Type: component.ConstructorInputType,
			Tag:  reflect.StructTag(component.Tag),
		}
	}

	return fieldList
}

func (p *registry) makeAllConfigStruct() reflect.Type {
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

	numFields := p.businessConfigType.NumField()
	fieldList := make([]reflect.StructField, 0, numFields+1)
	fieldList = append(fieldList, reflect.StructField{
		Name:      componentConfigFieldName,
		Type:      reflect.StructOf(p.makeComponentsConfigFields()),
		Tag:       `yaml:"base" json:"base" toml:"base"`,
		Anonymous: true,
	})
	for i := 0; i < numFields; i++ {
		fieldList = append(fieldList, p.businessConfigType.Field(i))
	}

	return reflect.StructOf(fieldList)
}

func (p *registry) loadComponents() {
	p.loadComponentsOnce.Do(func() {
		// app
		p.AddComponent(ComponentApp, func(context.Context, string, ...utils.OptionExtender) func() { return nil },
			WithTag("yaml", "app"), WithTag("json", "app"), WithTag("toml", "app"))

		// debug
		p.AddComponent(ComponentDebug, func(context.Context, bool, ...utils.OptionExtender) func() { return nil },
			WithTag("yaml", "debug"), WithTag("json", "debug"), WithTag("toml", "debug"))

		// crypto
		p.AddComponent(ComponentCrypto, CryptoConstruct,
			WithTag("yaml", "crypto"), WithTag("json", "crypto"), WithTag("toml", "crypto"))

		for _, item := range getComponents() {
			p.AddComponent(item.name, item.constructor, item.opt...)
		}

		/* example */
		// registry.AddComponent("ComponentExample", func(context.Context, string) func() { return nil },
		//    WithTag("custom_tag", "val"), WithTag("yaml", "val"))
	})
}

func (p *registry) initAllConfigByLoadFunc(businessConfig any, loadFn loadConfigFunc, opts ...utils.OptionExtender) {
	businessConfigVal := reflect.ValueOf(businessConfig)
	p.businessConfigType = utils.IndirectType(businessConfigVal.Type())

	p.businessConfig = reflect.New(p.businessConfigType).Interface()
	p.componentConfigs = reflect.New(p.makeComponentsConfigStruct()).Interface()
	if loadFn != nil {
		loadFn(p.businessConfig, opts...)
		loadFn(p.componentConfigs, opts...)
	}
}

func (p *registry) initComponents(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	ctxVal := reflect.ValueOf(ctx)
	o1 := reflect.ValueOf(utils.OptionExtender(AppName(p.appName)))
	o2 := reflect.ValueOf(utils.OptionExtender(DI(p.di)))

	baseObject := p.getBaseObject()
	destructors := make([]reflect.Value, 0, len(p.componentList))
	componentNames := make([]string, 0, len(p.componentList))
	hasCallbackComponentNames := make([]string, 0, len(p.componentList))
	for i := 0; i < len(p.componentList); i++ {
		com := p.componentList[i]
		comArgs := reflect.ValueOf(clone.Clone(baseObject.FieldByName(com.Name).Interface()))
		componentNames = append(componentNames, com.Name)
		if out := com.Constructor.Call([]reflect.Value{ctxVal, comArgs, o1, o2}); len(out) > 0 && !out[0].IsNil() {
			destructors = append(destructors, out[0])
			hasCallbackComponentNames = append(hasCallbackComponentNames, com.Name)
		}
	}

	/* print summary to stdout */
	pid := syscall.Getpid()
	app := p.AppName()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	log.Printf("%v [Gofusion] %s initialized total %d components below: %s\n",
		pid, app, len(componentNames), strings.Join(componentNames, ", "))

	once := new(sync.Once)
	return func() {
		once.Do(func() {
			initLocker.Lock()
			defer initLocker.Unlock()

			defer close(p.closeCh)

			p.initWg.Done()
			p.initWg.Wait()
			cancel()
			for i := len(destructors) - 1; i >= 0; i-- {
				log.Printf("%v [Gofusion] %s %s exiting...", pid, app, hasCallbackComponentNames[i])
				destructors[i].Call(nil)
				log.Printf("%v [Gofusion] %s %s exited", pid, app, hasCallbackComponentNames[i])
			}

			p.di.Clear()
			p.businessConfig = nil
			p.componentConfigs = nil
			p.initOnce = new(sync.Once)
		})
	}
}

func (p *registry) addComponent(com *Component) {
	firstNonCoreComIndex := -1
	for i, cp := range p.componentList {
		if !cp.isCore {
			firstNonCoreComIndex = i
			break
		}
	}
	if !com.isCore || firstNonCoreComIndex == -1 {
		p.componentList = append(p.componentList, com)
		sort.SliceStable(p.componentList, func(i, j int) bool {
			// core component would not be sorted
			if p.componentList[i].isCore || p.componentList[j].isCore {
				return false
			}

			orderA := indexComponent(p.componentList[i].Name)
			if orderA == -1 {
				return false
			}
			orderB := indexComponent(p.componentList[j].Name)
			if orderB == -1 {
				return true
			}

			return orderA < orderB
		})

		return
	}
	list := make([]*Component, len(p.componentList)+1)
	for i := range list {
		if i < firstNonCoreComIndex {
			list[i] = p.componentList[i]
		} else if i == firstNonCoreComIndex {
			list[i] = com
		} else {
			list[i] = p.componentList[i-1]
		}
	}

	p.componentList = list
}

func (p *registry) checkBusinessConfig(businessConfig any) {
	typ := reflect.TypeOf(businessConfig)
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Ptr {
		panic(errors.New("businessConfig should be a **struct"))
	}
}
