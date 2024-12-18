package redis

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "redis-conf", "", "", "json string for redis config")
}
