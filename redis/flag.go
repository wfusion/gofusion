package redis

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "redis", "", "", "json string for redis config")
}
