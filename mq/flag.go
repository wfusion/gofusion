package mq

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "mq", "", "", "json string for message queue config")
}
