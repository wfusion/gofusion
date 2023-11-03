package i18n

import (
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/common/utils"
)

type Localizable[T comparable] interface {
	AddMessages(code T, trans map[language.Tag]*Message, opts ...utils.OptionExtender) Localizable[T]
	Localize(code T, opts ...utils.OptionExtender) (message string)
}

// Conf i18n configure
type Conf struct {
	DefaultLang string `yaml:"default_lang" json:"default_lang" toml:"default_lang" default:"zh"`
}

// Message is a string that can be localized.
type Message struct {
	// ID uniquely identifies the message.
	ID string

	// Hash uniquely identifies the content of the message
	// that this message was translated from.
	Hash string

	// Description describes the message to give additional
	// context to translators that may be relevant for translation.
	Description string

	// LeftDelim is the left Go template delimiter.
	LeftDelim string

	// RightDelim is the right Go template delimiter.``
	RightDelim string

	// Zero is the content of the message for the CLDR plural form "zero".
	Zero string

	// One is the content of the message for the CLDR plural form "one".
	One string

	// Two is the content of the message for the CLDR plural form "two".
	Two string

	// Few is the content of the message for the CLDR plural form "few".
	Few string

	// Many is the content of the message for the CLDR plural form "many".
	Many string

	// Other is the content of the message for the CLDR plural form "other".
	Other string
}
