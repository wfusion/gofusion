package db

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "db-conf", "", "", "json string for database config")
}
