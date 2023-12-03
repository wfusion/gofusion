package lock

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "lock-config", "", "", "json string for lock config")
}
