package log

import (
	"context"
	"crypto/md5"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/infra/rotatelog"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}
	for name, conf := range confs {
		addInstance(ctx, name, conf, opt)
	}

	if opt.AppName == "" && appInstances[opt.AppName] != nil {
		if conf, ok := appInstances[opt.AppName][config.DefaultInstanceKey]; !ok || conf == nil {
			panic(ErrDefaultLoggerNotFound)
		}
	}

	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()
		if appInstances != nil {
			for _, instance := range appInstances[opt.AppName] {
				instance.flush()
			}
			delete(appInstances, opt.AppName)
		}

		// there maybe some locally logging, avoid some NPE crash as possible as we can do
		colorful := false
		if opt.AppName == "" {
			if confs != nil && confs[config.DefaultInstanceKey] != nil {
				colorful = confs[config.DefaultInstanceKey].ConsoleOutputOption.Colorful
			}
			globalLogger = defaultLogger(colorful)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	if !conf.EnableFileOutput && !conf.EnableConsoleOutput {
		panic(ErrUnknownOutput)
	}

	var cores []zapcore.Core
	if conf.EnableConsoleOutput {
		cfg := getEncoderConfig(conf)
		if conf.ConsoleOutputOption.Colorful {
			cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder := getEncoder(conf.ConsoleOutputOption.Layout, cfg)
		writer := zapcore.Lock(os.Stdout)
		logLevel := newZapLogLevel(opt.AppName, name, "enable_console_output", "log_level")
		cores = append(cores, zapcore.NewCore(encoder, writer, logLevel))
	}
	if conf.EnableFileOutput {
		var (
			ext     = ".log"
			logName string
		)

		cfg := getEncoderConfig(conf)
		encoder := getEncoder(conf.FileOutputOption.Layout, cfg)
		utils.IfAny(
			func() bool { logName = conf.FileOutputOption.Name; return utils.IsStrNotBlank(logName) },
			func() bool { logName = config.Use(opt.AppName).AppName() + ext; return utils.IsStrNotBlank(logName) },
			func() bool {
				logName = filepath.Base(env.WorkDir) + ext
				return logName != constant.PathSeparator+".log"
			},
			func() bool {
				sum := md5.Sum([]byte(env.WorkDir))
				logName = string(sum[:]) + ext
				return true
			},
		)

		rotationSize, err := humanize.ParseBytes(conf.FileOutputOption.RotationSize)
		if err != nil {
			panic(errors.Errorf("log component parse ratation size %s failed for name %s: %s",
				conf.FileOutputOption.RotationSize, name, err))
		}

		maxAge, err := time.ParseDuration(conf.FileOutputOption.RotationMaxAge)
		if err != nil {
			panic(errors.Errorf("log component parse ratation time %s failed for name %s: %s",
				conf.FileOutputOption.RotationMaxAge, name, err))
		}

		writer := zapcore.AddSync(zapcore.Lock(&rotatelog.Logger{
			Filename:   path.Join(filepath.Clean(conf.FileOutputOption.Path), logName),
			MaxSize:    rotationSize,
			MaxBackups: conf.FileOutputOption.RotationCount,
			MaxAge:     maxAge,
			Compress:   conf.FileOutputOption.Compress,
		}))
		logLevel := newZapLogLevel(opt.AppName, name, "enable_file_output", "log_level")
		cores = append(cores, zapcore.NewCore(encoder, writer, logLevel))
	}

	zopts := []zap.Option{
		zap.Hooks(),
		zap.AddCaller(),
		zap.AddStacktrace(newZapLogLevel(opt.AppName, name, "enable_file_output", "stacktrace_level")),
	}
	if config.Use(opt.AppName).Debug() {
		zopts = append(zopts, zap.Development())
	}

	zapLogger := zap.
		New(zapcore.NewTee(cores...), zopts...).
		Named(config.Use(opt.AppName).AppName())

	fusLogger := &logger{name: name, logger: zapLogger, sugaredLogger: zapLogger.Sugar()}

	rwlock.Lock()
	defer rwlock.Unlock()
	if appInstances == nil {
		appInstances = make(map[string]map[string]*logger)
	}
	if appInstances[opt.AppName] == nil {
		appInstances[opt.AppName] = make(map[string]*logger)
	}
	if _, ok := appInstances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	appInstances[opt.AppName][name] = fusLogger

	if opt.AppName == "" && name == config.DefaultInstanceKey {
		globalLogger = fusLogger
		if opt.App != nil {
			opt.App.Options(fx.WithLogger(func() fxevent.Logger {
				return &fxevent.ZapLogger{Logger: fusLogger.logger}
			}))
		}
	}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() Loggable { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}
	if opt.App != nil {
		opt.App.MustProvide(
			func() Loggable { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}
}

func init() {
	config.AddComponent(config.ComponentLog, Construct, config.WithFlag(&flagString))
}
