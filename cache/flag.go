package cache

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "cache", "", "", "json string for cache config")
}
