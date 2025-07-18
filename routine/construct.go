package routine

import (
	"context"
	"log"
	"reflect"
	"syscall"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"

	fusLog "github.com/wfusion/gofusion/log"

	_ "github.com/wfusion/gofusion/log/customlogger"
)

const (
	defaultMaxPoolSize = 10000000
)

func Construct(ctx context.Context, conf Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[candyOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}

	if conf.MaxRoutineAmount <= 0 {
		conf.MaxRoutineAmount = defaultMaxPoolSize
	}

	rwlock.Lock()
	defer rwlock.Unlock()
	if pools == nil {
		pools = make(map[string]map[string]Pool)
	}
	if pools[opt.AppName] == nil {
		pools[opt.AppName] = make(map[string]Pool)
	}
	if ignored == nil {
		ignored = make(map[string]*atomic.Int64)
	}
	if ignored[opt.AppName] == nil {
		ignored[opt.AppName] = atomic.NewInt64(0)
	}
	if idles == nil {
		idles = make(map[string]*atomic.Int64)
	}
	if idles[opt.AppName] == nil {
		idles[opt.AppName] = atomic.NewInt64(int64(conf.MaxRoutineAmount))
	}
	if utils.IsStrNotBlank(conf.Logger) {
		if defaultLogger == nil {
			defaultLogger = make(map[string]ants.Logger)
		}
		if defaultLogger[opt.AppName] == nil {
			logger := reflect.New(inspect.TypeOf(conf.Logger)).Interface().(ants.Logger)
			defaultLogger[opt.AppName] = logger
			if custom, ok := logger.(customLogger); ok {
				l := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
				custom.Init(l, opt.AppName)
			}
		}
	}
	maxReleaseTime := conf.MaxReleaseTimePerPool.Duration

	go startDaemonRoutines(ctx, opt.AppName, &conf)

	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		allExited := func() bool {
			return idles[opt.AppName].Load() == int64(conf.MaxRoutineAmount)-ignored[opt.AppName].Load()
		}

		// waiting for pool
		if pools != nil {
			for name, pool := range pools[opt.AppName] {
				if err := pool.ReleaseTimeout(maxReleaseTime, ignoreMutex()); err != nil {
					log.Printf("%v [Gofusion] %s %s exit with releasing pool %s failed because %s",
						pid, app, config.ComponentGoroutinePool, name, err)
				}
				delete(pools[opt.AppName], name)
			}
		}

		log.Printf("%v [Gofusion] %s %s pool routines are recycled", pid, app, config.ComponentGoroutinePool)

		// waiting for go
		utils.Timeout(maxReleaseTime, utils.TimeoutWg(&wg))
		log.Printf("%v [Gofusion] %s %s go routines are recycled", pid, app, config.ComponentGoroutinePool)

		if !allExited() {
			log.Printf("%v [Gofusion] %s %s exit without all goroutines recycled [exists%v]",
				pid, app, config.ComponentGoroutinePool, showRoutine(opt.AppName))
		}

		delete(ignored, opt.AppName)
		delete(idles, opt.AppName)
		delete(routines, opt.AppName)
	}
}

func configs(appName string) (conf Conf) {
	_ = config.Use(appName).LoadComponentConfig(config.ComponentGoroutinePool, &conf)
	return
}

func forceSync(appName string) bool {
	return configs(appName).ForceSync
}

func init() {
	config.AddComponent(config.ComponentGoroutinePool, Construct, config.WithFlag(&flagString))
}
