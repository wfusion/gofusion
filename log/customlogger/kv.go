package customlogger

import (
	"strings"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	kvFields = log.Fields{"component": strings.ToLower(config.ComponentKV)}
)
