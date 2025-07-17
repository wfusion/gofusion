package trace

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "trace", "", "", "json string for trace config")
}
