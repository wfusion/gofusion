package db

import (
	"context"
	"log"
	"syscall"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/routine"
)

func startDaemonRoutines(ctx context.Context, appName, name string) {
	ticker := time.Tick(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s metrics exited", syscall.Getpid(), config.ComponentDB, name)
			return
		case <-ticker:
			routine.Loop(metricDBConn, routine.Args(ctx, appName, name), routine.AppName(appName))
			routine.Loop(metricDBLatency, routine.Args(ctx, appName, name), routine.AppName(appName))
		}
	}
}

func metricDBConn(ctx context.Context, appName, name string) {
	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()

		_, err := instances[appName][name].GetProxy().DB()
		if err != nil {
			panic(err)
		}

		// TODO: emit metrics
		// idle := sqlDB.Stats().Idle
		// inUse := sqlDB.Stats().InUse
	})
}

func metricDBLatency(ctx context.Context, appName, name string) {
	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()
		sqlDB := utils.Must(instances[appName][name].GetProxy().DB())

		// begin := time.Now()
		if err := sqlDB.Ping(); err == nil {
			// TODO: emit metrics
		}
	})
}
