package config

import (
	"flag"

	"github.com/spf13/pflag"
)

var (
	debugFlag        bool
	appFlagString    string
	appBizFlagString string
	customConfigPath []string
)

func init() {
	pflag.StringSliceVar(&customConfigPath, "config-file", nil, "specify config file path, e.g. configs/app.yml")
	pflag.StringVarP(&appFlagString, "app", "", "", "app name")
	pflag.BoolVarP(&debugFlag, "debug", "", false,
		"enable debug mode, only works for http and db component now")
	pflag.StringVarP(&appBizFlagString, "app-config", "", "", "json string for app config")
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
}

func parseFlags() {
	if !pflag.Parsed() {
		pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
		pflag.Parse()
	}
}
