package metrics

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "metrics-config", "", "", "json string for metrics config")
}
