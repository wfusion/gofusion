package cron

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "cron-config", "", "", "json string for cron config")
}
