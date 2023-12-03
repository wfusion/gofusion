package redis

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "redis-config", "", "", "json string for redis config")
}
