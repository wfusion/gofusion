package async

import "github.com/spf13/pflag"

var flagString string

func init() {
	pflag.StringVarP(&flagString, "async", "", "", "json string for async config")
}
