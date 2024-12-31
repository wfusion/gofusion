package cron

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "cron", "", "", "json string for cron config")
}
