package routine

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "goroutine-config", "", "", "json string for goroutine config")
}
