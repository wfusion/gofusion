package http

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "http", "", "", "json string for http config")
}
