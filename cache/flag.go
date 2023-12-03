package cache

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "cache-config", "", "", "json string for cache config")
}
