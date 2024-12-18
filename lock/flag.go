package lock

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "lock-conf", "", "", "json string for lock config")
}
