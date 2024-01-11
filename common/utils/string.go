package utils

import (
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/wfusion/gofusion/common/constant"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func IsStrBlank(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return !(unicode.IsSpace(r)) }) < 0
}

func IsStrPtrBlank(s *string) bool {
	return s == nil || IsStrBlank(*s)
}

func IsStrNotBlank(s string) bool {
	return !IsStrBlank(s)
}

func IsStrPtrNotBlank(s *string) bool {
	return s != nil && IsStrNotBlank(*s)
}

var (
	keywordFuzzyDelimited = []string{
		constant.Space,
		constant.Colon,
		constant.Hyphen,
		constant.Underline,
	}
)

func FuzzyKeyword(keyword string) []string {
	words := strings.Fields(constant.NonNumberLetterReg.ReplaceAllString(keyword, " "))
	compact := strings.Join(words, "")
	lowerWords := SliceMapping(words, func(s string) string { return strings.ToLower(s) })
	upperWords := SliceMapping(words, func(s string) string { return strings.ToUpper(s) })
	titleWords := SliceMapping(words, func(s string) string { return cases.Title(language.English).String(s) })

	s := NewSet(keyword)
	s.Insert(
		compact,
		strings.ToUpper(compact),
		strings.ToLower(compact),
		strcase.ToCamel(keyword),
		strcase.ToLowerCamel(keyword),
		strcase.ToKebab(keyword),
		strcase.ToSnake(keyword),
		strcase.ToScreamingSnake(keyword),
		strcase.ToScreamingKebab(keyword),
	)
	for _, delimited := range keywordFuzzyDelimited {
		s.Insert(
			strings.Join(lowerWords, delimited),
			strings.Join(upperWords, delimited),
			strings.Join(titleWords, delimited),
		)
	}

	return s.Items()
}

func init() {
	strcase.ConfigureAcronym("I18n", "i18n")
}
