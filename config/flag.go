package config

import (
	"flag"

	"github.com/spf13/pflag"
)

const (
	appFlagKey   = "app"
	debugFlagKey = "debug"
)

var (
	debugFlag              bool
	appFlagString          string
	appBizFlagString       string
	customConfigPath       []string
	remoteConfigFlagString string
)

func init() {
	pflag.StringSliceVarP(&customConfigPath, "conf", "", nil, "specify config file path, e.g. appConfigs/app.yml")
	pflag.StringVarP(&appFlagString, appFlagKey, "", "", "app name")
	pflag.BoolVarP(&debugFlag, debugFlagKey, "", false,
		"enable debug mode, only works for http and db component now")
	pflag.StringVarP(&appBizFlagString, "app-conf", "", "", "json string for app config")
	pflag.StringVarP(&remoteConfigFlagString, "remote-config", "", "", "json string for configuration center config")
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
}

func parseFlags() {
	if !pflag.Parsed() {
		pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
		pflag.Parse()
	}
}
