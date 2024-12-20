package customlogger

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/log"
)

func FxWithLoggerProvider(log log.Loggable) func() fxevent.Logger {
	logger := inspect.GetField[*zap.Logger](log, "logger")
	return func() fxevent.Logger {
		return &fxevent.ZapLogger{Logger: logger}
	}
}
