package routine

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "routine", "", "", "json string for goroutine config")
}
