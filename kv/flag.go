package kv

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "kv", "", "", "json string for kv config")
}
