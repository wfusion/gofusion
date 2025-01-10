package encoder

import (
	"go.uber.org/zap/zapcore"

	"github.com/wfusion/gofusion/common/utils"
)

var (
	SkipCallers = []string{
		"*github.com/wfusion/gofusion*/log/candy.go",
		"*github.com/wfusion/gofusion*/log/logger.go",
		"*github.com/wfusion/gofusion*/log/customlogger/*.go",
		"*github.com/wfusion/gofusion*/db/tx.go",
		"*github.com/wfusion/gofusion*/db/dal.go",
		"*github.com/wfusion/gofusion*/db/candy.go",
		"*github.com/wfusion/gofusion*/db/plugins/*.go",
		"*github.com/wfusion/gofusion*/db/callbacks/*.go",
		"*github.com/wfusion/gofusion*/db/softdelete/*.go",
		"*github.com/wfusion/gofusion*/kv/*.go",
		"*github.com/wfusion/gofusion*/cron/log.go",
		"*github.com/wfusion/gofusion*/async/log.go",
		"*github.com/wfusion/gofusion*/mq/log.go",
		"*github.com/wfusion/gofusion*/mq/mq.go",
		"*github.com/wfusion/gofusion*/routine/*.go",
		"*github.com/wfusion/gofusion*/common/infra/asynq/*.go",
		"*github.com/wfusion/gofusion*/common/infra/watermill/log.go",
	}

	// Positions in the call stack when tracing to report the calling method
	minimumCallerDepthOpt         utils.OptionExtender
	defaultSuffixedRegOpt         utils.OptionExtender
	defaultSuffixesRegPatternList = []string{
		`go.uber.org/zap(|@v.*)/.*go$`,
		`gorm.io/gorm(|@v.*)/.*go$`,
		`mysql@v.*/.*go$`,
		`postgres@v.*/.*go`,
		`sqlserver@v.*/.*go`,
		`mongo-driver(|@v.*)/.*go$`,
		`github.com/redis/go-redis/v9(|@v.*)/.*go$`,
		`asm_(amd64|arm64)\.s$`,
	}
)

const (
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 7 // should be github.com/wfusion/gofusion/log/candy.go:14
)

func init() {
	defaultSuffixedRegOpt = utils.SkipRegexps(defaultSuffixesRegPatternList...)
	minimumCallerDepthOpt = utils.SkipKnownDepth(knownLogrusFrames)
}

// SkipCallerEncoder skip custom stack when encoding stack
func SkipCallerEncoder(skipSuffixed []string, shorter bool) func(
	caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	skipGlobOpt := utils.SkipGlobs(skipSuffixed...)
	return func(entryCaller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		caller := utils.GetCaller(
			maximumCallerDepth,
			skipGlobOpt,
			defaultSuffixedRegOpt,
			minimumCallerDepthOpt,
		)

		entryCaller.PC = caller.PC
		entryCaller.File = caller.File
		entryCaller.Line = caller.Line
		entryCaller.Function = caller.Function
		if shorter {
			enc.AppendString(entryCaller.TrimmedPath())
		} else {
			enc.AppendString(entryCaller.FullPath())
		}
	}
}
