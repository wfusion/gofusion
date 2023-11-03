package i18n

import (
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

var (
	Bundle *bundle[int]

	locker               sync.RWMutex
	defaultLang          = language.Chinese
	defaultErrorMessages = map[language.Tag]string{
		language.Chinese: "系统错误，请稍后重试，如需帮助可联系您的客户经理！",
		language.English: "System error! Please try again later and contact your account manager if you need help!",
	}
)

type bundle[T comparable] struct {
	dup    *utils.Set[T]
	bundle *i18n.Bundle

	vars  map[T][]string
	mutex sync.RWMutex
}

func NewBundle[T comparable](lang language.Tag) Localizable[T] {
	b := &bundle[T]{
		dup:    utils.NewSet[T](),
		bundle: i18n.NewBundle(lang),
		vars:   make(map[T][]string),
	}

	b.bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	b.bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	b.bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	b.bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)

	return b
}

type addMessagesOption struct {
	vars []string
}

func Var(vars ...string) utils.OptionFunc[addMessagesOption] {
	return func(o *addMessagesOption) {
		o.vars = vars
	}
}

func (i *bundle[T]) AddMessages(code T, trans map[language.Tag]*Message, opts ...utils.OptionExtender) Localizable[T] {
	o := utils.ApplyOptions[addMessagesOption](opts...)

	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.checkDuplicated(code, trans)

	id := cast.ToString(code)
	i.vars[code] = o.vars
	for lang, msg := range trans {
		i.bundle.MustAddMessages(lang, &i18n.Message{
			ID:          id,
			Hash:        msg.Hash,
			Description: msg.Description,
			LeftDelim:   msg.LeftDelim,
			RightDelim:  msg.RightDelim,
			Zero:        msg.Zero,
			One:         msg.One,
			Two:         msg.Two,
			Few:         msg.Few,
			Many:        msg.Many,
			Other:       msg.Other,
		})
	}
	return i
}

func (i *bundle[T]) checkDuplicated(code T, trans map[language.Tag]*Message) {
	if !i.dup.Contains(code) {
		i.dup.Insert(code)
		return
	}
	if trans == nil {
		panic(errors.Errorf("%+v %s code translation is empty", code, cast.ToString(code)))
	}

	// panic if duplicated
	var (
		cfg                = &i18n.LocalizeConfig{MessageID: cast.ToString(code)}
		existMsgEn         = i18n.NewLocalizer(i.bundle, language.English.String()).MustLocalize(cfg)
		existMsgCn         = i18n.NewLocalizer(i.bundle, language.Chinese.String()).MustLocalize(cfg)
		dupMsgCn, dupMsgEn string
	)
	if dupMsg, ok := trans[language.Chinese]; ok {
		dupMsgCn = dupMsg.Other
	}
	if dupMsg, ok := trans[language.English]; ok {
		dupMsgEn = dupMsg.Other
	}

	panic(errors.Errorf("%s(%s)(%+v)(%v) is duplicated with %s(%s)",
		dupMsgCn, dupMsgEn, code, cast.ToString(code), existMsgCn, existMsgEn))
}

type localizeOption struct {
	lang         language.Tag
	langs        []string
	pluralCount  any
	templateData map[string]any
}

func Param(data map[string]any) utils.OptionFunc[localizeOption] {
	return func(o *localizeOption) {
		o.templateData = data
	}
}

func Plural(pluralCount any) utils.OptionFunc[localizeOption] {
	return func(o *localizeOption) {
		o.pluralCount = pluralCount
	}
}

func Lang(lang language.Tag) utils.OptionFunc[localizeOption] {
	return func(o *localizeOption) {
		o.lang = lang
	}
}

func Langs(langs []string) utils.OptionFunc[localizeOption] {
	return func(o *localizeOption) {
		if len(langs) > 0 {
			o.lang, _ = language.Parse(langs[0])
		}
		o.langs = clone.SliceComparable(langs)
	}
}

func (i *bundle[T]) Localize(code T, opts ...utils.OptionExtender) (message string) {
	option := utils.ApplyOptions[localizeOption](opts...)
	if option.templateData == nil && len(i.vars) > 0 {
		option.templateData = make(map[string]any, len(i.vars))
	}

	// TODO: Access the third-party internationalization platform to obtain text
	cfg := &i18n.LocalizeConfig{
		MessageID:    cast.ToString(code),
		TemplateData: option.templateData,
		PluralCount:  option.pluralCount,
	}

	i.mutex.RLock()
	defer i.mutex.RUnlock()
	// Assign an empty string to a variable to avoid rendering < no value > data
	for _, v := range i.vars[code] {
		if _, ok := option.templateData[v]; !ok {
			option.templateData[v] = ""
		}
	}
	message, err := i18n.NewLocalizer(i.bundle, option.langs...).Localize(cfg)
	if err == nil {
		return
	}
	message, ok := defaultErrorMessages[option.lang]
	if ok {
		return
	}

	return defaultErrorMessages[defaultLang]
}
