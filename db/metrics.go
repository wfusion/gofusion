package db

import (
	"context"
	"log"
	"syscall"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsPoolIdleKey         = []string{"db", "idle"}
	metricsPoolTotalKey        = []string{"db", "total"}
	metricsPoolInUseKey        = []string{"db", "inuse"}
	metricsPoolWaitCountKey    = []string{"db", "wait", "count"}
	metricsPoolWaitDurationKey = []string{"db", "wait", "duration"}
	metricsLatencyKey          = []string{"db", "latency"}
	metricsLatencyBuckets      = []float64{
		.1, .25, .5, .75, .90, .95, .99,
		1, 2.5, 5, 7.5, 9, 9.5, 9.9,
		10, 25, 50, 75, 90, 95, 99,
	}
)

func startDaemonRoutines(ctx context.Context, appName, name string, conf *Conf) {
	ticker := time.Tick(time.Second * 5)
	app := config.Use(appName).AppName()
	labels := []metrics.Label{
		{Key: "config", Value: name},
		{Key: "database", Value: conf.DB},
	}
	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentDB, name)
			return
		case <-ticker:
			go metricDBStats(ctx, appName, name, labels)
			go metricDBLatency(ctx, appName, name, labels)
		}
	}
}

func metricDBStats(ctx context.Context, appName, name string, labels []metrics.Label) {
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
		db := instance.GetProxy()
		sqlDB, err := db.DB()
		if err != nil {
			return
		}

		idleKey := append([]string{appName}, metricsPoolIdleKey...)
		inuseKey := append([]string{appName}, metricsPoolInUseKey...)
		totalKey := append([]string{appName}, metricsPoolTotalKey...)
		waitCountKey := append([]string{appName}, metricsPoolWaitCountKey...)
		waitDurationKey := append([]string{appName}, metricsPoolWaitDurationKey...)

		stats := sqlDB.Stats()
		waitDuration := float64(stats.WaitDuration) / float64(time.Millisecond)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.SetGauge(ctx, idleKey, float64(stats.Idle), metrics.Labels(labels))
					m.SetGauge(ctx, inuseKey, float64(stats.InUse), metrics.Labels(labels))
					m.SetGauge(ctx, totalKey, float64(stats.OpenConnections), metrics.Labels(labels))
					m.SetGauge(ctx, waitCountKey, float64(stats.WaitCount), metrics.Labels(labels))
					m.SetGauge(ctx, waitDurationKey, waitDuration, metrics.Labels(labels))
				} else {
					m.SetGauge(ctx, metricsPoolIdleKey, float64(stats.Idle), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolInUseKey, float64(stats.InUse), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolTotalKey, float64(stats.OpenConnections), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolWaitCountKey, float64(stats.WaitCount), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolWaitDurationKey, waitDuration, metrics.Labels(labels))
				}
			}
		}
	})
}

func metricDBLatency(ctx context.Context, appName, name string, labels []metrics.Label) {
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
		db := instance.GetProxy()
		sqlDB, err := db.DB()
		if err != nil {
			return
		}

		begin := time.Now()
		if err := sqlDB.Ping(); err != nil {
			return
		}
		latency := float64(time.Since(begin)) / float64(time.Millisecond)
		latencyKey := append([]string{appName}, metricsLatencyKey...)
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
