package i18n

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "i18n", "", "", "json string for i18n config")
}
