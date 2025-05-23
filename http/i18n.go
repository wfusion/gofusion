package http

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"

	ut "github.com/go-playground/universal-translator"
	enT "github.com/go-playground/validator/v10/translations/en"
	zhT "github.com/go-playground/validator/v10/translations/zh"
)

var (
	I18n  i18n.Localizable[Errcode]
	i18ns map[string]i18n.Localizable[Errcode]

	I18nErr  i18n.Localizable[Error]
	i18nErrs map[string]i18n.Localizable[Error]

	ginBindingI18nOnce       = new(sync.Once)
	ginBindingI18nTranslator ut.Translator
)

func Localizable(opts ...utils.OptionExtender) i18n.Localizable[Errcode] {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	i, ok := i18ns[opt.appName]
	if !ok {
		panic(errors.Errorf("http i18n not founc: %s", opt.appName))
	}
	return i
}

func LocalizableError(opts ...utils.OptionExtender) i18n.Localizable[Error] {
	opt := utils.ApplyOptions[useOption](opts...)

	locker.RLock()
	defer locker.RUnlock()
	i, ok := i18nErrs[opt.appName]
	if !ok {
		panic(errors.Errorf("http i18n not founc: %s", opt.appName))
	}
	return i
}

func ginBindingValidatorI18n(appName string) {
	ginBindingI18nOnce.Do(func() {
		var ok bool
		engine, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			return
		}

		enLocales := en.New()
		zhLocales := zh.New()
		switch lang := i18n.DefaultLang(i18n.AppName(appName)); lang {
		case language.English:
			ginBindingI18nTranslator, ok = ut.New(enLocales, zhLocales, enLocales).GetTranslator(lang.String())
			if !ok {
				return
			}
			utils.MustSuccess(enT.RegisterDefaultTranslations(engine, ginBindingI18nTranslator))
		default:
			ginBindingI18nTranslator, ok = ut.New(zhLocales, zhLocales, enLocales).GetTranslator(lang.String())
			if !ok {
				return
			}
			utils.MustSuccess(zhT.RegisterDefaultTranslations(engine, ginBindingI18nTranslator))
		}
	})
}

func parseGinBindingValidatorError(src error) (dst error) {
	e, ok := src.(validator.ValidationErrors)
	if !ok || ginBindingI18nTranslator == nil {
		return src
	}
	return errors.New(strings.Join(utils.MapValues(e.Translate(ginBindingI18nTranslator)), constant.LineBreak))
}
