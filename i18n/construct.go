package i18n

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"github.com/wfusion/gofusion/config"
)

func Construct(ctx context.Context, conf Conf, opts ...utils.OptionExtender) func() {
	var err error
	lang := defaultLang
	if utils.IsStrNotBlank(conf.DefaultLang) {
		if lang, err = language.Parse(conf.DefaultLang); err != nil {
			panic(err)
		}
	}

	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}

	locker.Lock()
	defer locker.Unlock()
	if Bundle == nil {
		Bundle = &bundle[int]{
			dup:    utils.NewSet[int](),
			bundle: i18n.NewBundle(lang),
		}
		Bundle.bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
		Bundle.bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		Bundle.bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
		Bundle.bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	}

	// ioc
	if opt.DI != nil {
		opt.DI.
			MustProvide(func() Localizable[int] { return NewBundle[int](lang) }).
			MustProvide(func() Localizable[string] { return NewBundle[string](lang) })
	}

	return func() {
		locker.Lock()
		defer locker.Unlock()
		Bundle = &bundle[int]{
			dup:    utils.NewSet[int](),
			bundle: i18n.NewBundle(lang),
		}
	}
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func DefaultLang(opts ...utils.OptionExtender) (lang language.Tag) {
	opt := utils.ApplyOptions[useOption](opts...)

	conf := new(Conf)
	utils.MustSuccess(config.Use(opt.appName).LoadComponentConfig(config.ComponentI18n, conf))
	return utils.Must(language.Parse(conf.DefaultLang))
}

func init() {
	config.AddComponent(config.ComponentI18n, Construct)
}
