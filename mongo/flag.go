package mongo

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "mongo", "", "", "json string for mongo config")
}
