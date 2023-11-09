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
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/env"
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
	logLevel := getLogLevel(conf.LogLevel)

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
			func() bool { logName = filepath.Base(env.WorkDir) + ext; return logName != "/.log" },
			func() bool {
				sum := md5.Sum([]byte(env.WorkDir))
				logName = string(sum[:]) + ext
				return true
			},
		)

		rotationSize := cast.ToInt64(conf.FileOutputOption.RotationSize) * int64(humanize.MByte)
		if rotationSize == 0 {
			u, err := humanize.ParseBytes(conf.FileOutputOption.RotationSize)
			if err != nil {
				panic(errors.Errorf("log component parse ratation size %s failed for name %s: %s",
					conf.FileOutputOption.RotationSize, name, err))
			}
			rotationSize = int64(u)
		}
		if rotationSize < humanize.MByte {
			panic(errors.Errorf("log component %s parse ratation size %v bytes is smaller than 1m",
				name, rotationSize))
		}

		maxAge := time.Duration(cast.ToInt(conf.FileOutputOption.RotationTime)) * time.Hour
		if maxAge == 0 {
			d, err := time.ParseDuration(conf.FileOutputOption.RotationTime)
			if err != nil {
				panic(errors.Errorf("log component parse ratation time %s failed for name %s: %s",
					conf.FileOutputOption.RotationTime, name, err))
			}
			maxAge = d
		}
		if maxAge < 24*time.Hour {
			panic(errors.Errorf("log component %s parse ratation time %v is shorter than 24h",
				name, maxAge))
		}

		writer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   path.Join(conf.FileOutputOption.Path, logName),
			MaxSize:    int(rotationSize / int64(humanize.MByte)),
			MaxBackups: conf.FileOutputOption.RotationCount,
			MaxAge:     int(maxAge) / int(24*time.Hour),
			Compress:   conf.FileOutputOption.Compress,
		})
		cores = append(cores, zapcore.NewCore(encoder, writer, logLevel))
	}

	zapLogger := zap.
		New(
			zapcore.NewTee(cores...),
			zap.AddStacktrace(getLogLevel(conf.StacktraceLevel)),
			zap.AddCaller(),
			zap.Hooks(),
		).
		Named(config.Use(opt.AppName).AppName())

	fmkLogger := &logger{name: name, logger: zapLogger, sugaredLogger: zapLogger.Sugar()}

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
	appInstances[opt.AppName][name] = fmkLogger

	if name == config.DefaultInstanceKey {
		globalLogger = fmkLogger
	}

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(
			func() Logable { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}
}

func init() {
	config.AddComponent(config.ComponentLog, Construct)
}
