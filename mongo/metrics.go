package mongo

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
		app := config.Use(appName).AppName()
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentMongo, name)
			return
		case <-ticker:
			routine.Loop(metricMongoConn, routine.Args(ctx, appName, name), routine.AppName(appName))
			routine.Loop(metricMongoLatency, routine.Args(ctx, appName, name), routine.AppName(appName))
		}
	}
}

func metricMongoConn(ctx context.Context, appName, name string) {
	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()
		_ = instances[appName][name].GetProxy()

		// TODO: emit metrics
		// idle := sqlDB.Stats().Idle
		// inUse := sqlDB.Stats().InUse
	})
}

func metricMongoLatency(ctx context.Context, appName, name string) {
	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()
		mgo := instances[appName][name].GetProxy()

		// begin := time.Now()
		if err := mgo.Ping(ctx, nil); err == nil {
			// TODO: emit metrics
		}
	})
}
