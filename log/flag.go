package log

import (
	"github.com/spf13/pflag"
)

var flagString string

func init() {
	pflag.StringVarP(&flagString, "log", "", "", "json string for log config")
}
