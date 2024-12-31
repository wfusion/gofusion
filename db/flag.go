package db

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "db", "", "", "json string for database config")
}
