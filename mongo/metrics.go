package mongo

import (
	"context"
	"log"
	"sync"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsPoolIdleKey      = []string{"mongo", "idle"}
	metricsPoolInUseKey     = []string{"mongo", "inuse"}
	metricsPoolTotalKey     = []string{"mongo", "total"}
	metricsPoolLocker       = new(sync.RWMutex)
	metricsPoolInUseCounter = map[string]map[string]*atomic.Int64{}
	metricsPoolTotalCounter = map[string]map[string]*atomic.Int64{}
	metricsLatencyKey       = []string{"mongo", "latency"}
	metricsLatencyBuckets   = []float64{
		.1, .25, .5, .75, .90, .95, .99,
		1, 2.5, 5, 7.5, 9, 9.5, 9.9,
		10, 25, 50, 75, 90, 95, 99,
		100, 250, 500, 750, 900, 950, 990,
	}
)

func startDaemonRoutines(ctx context.Context, appName, name string, conf *Conf) {
	ticker := time.Tick(time.Second * 5)
	app := config.Use(appName).AppName()
	labels := []metrics.Label{
		{Key: "config", Value: name},
		{Key: "database", Value: conf.DB},
	}

	log.Printf("%v [Gofusion] %s %s %s metrics start", syscall.Getpid(), app, config.ComponentMongo, name)
	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentMongo, name)
			return
		case <-ticker:
			go metricMongoStats(ctx, appName, name, labels)
			go metricMongoLatency(ctx, appName, name, labels)
		}
	}
}

func metricMongoStats(ctx context.Context, appName, name string, labels []metrics.Label) {
	_, _ = utils.Catch(func() {
		var total, inuse int64
		_, err := utils.Catch(func() {
			metricsPoolLocker.RLock()
			defer metricsPoolLocker.RUnlock()
			inuse = metricsPoolInUseCounter[appName][name].Load()
			total = metricsPoolTotalCounter[appName][name].Load()
		})
		if err != nil {
			return
		}

		app := config.Use(appName).AppName()
		idleKey := append([]string{app}, metricsPoolIdleKey...)
		inuseKey := append([]string{app}, metricsPoolInUseKey...)
		totalKey := append([]string{app}, metricsPoolTotalKey...)
		ide := total - inuse
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.SetGauge(ctx, idleKey, float64(ide), metrics.Labels(labels))
					m.SetGauge(ctx, inuseKey, float64(inuse), metrics.Labels(labels))
					m.SetGauge(ctx, totalKey, float64(total), metrics.Labels(labels))
				} else {
					m.SetGauge(ctx, metricsPoolIdleKey, float64(ide), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolInUseKey, float64(inuse), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolTotalKey, float64(total), metrics.Labels(labels))
				}
			}
		}
	})
}

func metricMongoLatency(ctx context.Context, appName, name string, labels []metrics.Label) {
	select {
	case <-ctx.Done():
		return
	default:

	}

	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()
		instances, ok := appInstances[appName]
		if !ok {
			return
		}
		instance, ok := instances[name]
		if !ok {
			return
		}

		mgoCli := instance.GetProxy()
		begin := time.Now()
		if err := mgoCli.Ping(ctx, nil); err != nil {
			return
		}

		latency := float64(time.Since(begin)) / float64(time.Millisecond)
		latencyKey := append([]string{config.Use(appName).AppName()}, metricsLatencyKey...)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.AddSample(ctx, latencyKey, latency,
						metrics.Labels(labels),
						metrics.PrometheusBuckets(metricsLatencyBuckets),
					)
				} else {
					m.AddSample(ctx, metricsLatencyKey, latency,
						metrics.Labels(labels),
						metrics.PrometheusBuckets(metricsLatencyBuckets),
					)
				}
			}
		}
	})
}

func metricsPoolMonitor(appName, name string) func(evt *event.PoolEvent) {
	metricsPoolLocker.Lock()
	defer metricsPoolLocker.Unlock()
	if metricsPoolTotalCounter[appName] == nil {
		metricsPoolTotalCounter[appName] = make(map[string]*atomic.Int64)
	}
	if metricsPoolTotalCounter[appName][name] == nil {
		metricsPoolTotalCounter[appName][name] = atomic.NewInt64(0)
	}
	if metricsPoolInUseCounter[appName] == nil {
		metricsPoolInUseCounter[appName] = make(map[string]*atomic.Int64)
	}
	if metricsPoolInUseCounter[appName][name] == nil {
		metricsPoolInUseCounter[appName][name] = atomic.NewInt64(0)
	}

	inuse := metricsPoolInUseCounter[appName][name]
	total := metricsPoolTotalCounter[appName][name]
	return func(evt *event.PoolEvent) {
		switch evt.Type {
		case event.PoolCreated:
		case event.PoolReady:
		case event.PoolCleared:
		case event.PoolClosedEvent:
		case event.ConnectionCreated:
			total.Add(1)
		case event.ConnectionReady:
		case event.ConnectionClosed:
			total.Add(-1)
		case event.GetStarted:
		case event.GetFailed:
		case event.GetSucceeded:
			inuse.Add(1)
		case event.ConnectionReturned:
			inuse.Add(-1)
		}
	}

}
