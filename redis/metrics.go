package redis

import (
	"context"
	"log"
	"syscall"
	"time"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsPoolIdleKey    = []string{"redis", "idle"}
	metricsPoolTotalKey   = []string{"redis", "total"}
	metricsPoolStaleKey   = []string{"redis", "stale"}
	metricsPoolHitsKey    = []string{"redis", "hits"}
	metricsPoolMissesKey  = []string{"redis", "misses"}
	metricsLatencyKey     = []string{"redis", "latency"}
	metricsLatencyBuckets = []float64{
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
		{Key: "database", Value: cast.ToString(conf.DB)},
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentRedis, name)
			return
		case <-ticker:
			go metricRedisStats(ctx, appName, name, labels)
			go metricRedisLatency(ctx, appName, name, labels)
		}
	}
}

func metricRedisStats(ctx context.Context, appName, name string, labels []metrics.Label) {
	select {
	case <-ctx.Done():
		return
	default:

	}

	_, _ = utils.Catch(func() {
		rwlock.RLock()
		defer rwlock.RUnlock()
		_ = appInstances[appName][name].GetProxy()
		instances, ok := appInstances[appName]
		if !ok {
			return
		}
		instance, ok := instances[name]
		if !ok {
			return
		}

		idleKey := append([]string{appName}, metricsPoolIdleKey...)
		staleKey := append([]string{appName}, metricsPoolStaleKey...)
		totalKey := append([]string{appName}, metricsPoolTotalKey...)
		hitsKey := append([]string{appName}, metricsPoolHitsKey...)
		missesKey := append([]string{appName}, metricsPoolMissesKey...)

		rdsCli := instance.GetProxy()
		stats := rdsCli.PoolStats()
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.SetGauge(ctx, idleKey, float64(stats.IdleConns), metrics.Labels(labels))
					m.SetGauge(ctx, staleKey, float64(stats.StaleConns), metrics.Labels(labels))
					m.SetGauge(ctx, totalKey, float64(stats.TotalConns), metrics.Labels(labels))
					m.SetGauge(ctx, hitsKey, float64(stats.Hits), metrics.Labels(labels))
					m.SetGauge(ctx, missesKey, float64(stats.Misses), metrics.Labels(labels))
				} else {
					m.SetGauge(ctx, metricsPoolIdleKey, float64(stats.IdleConns), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolStaleKey, float64(stats.StaleConns), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolTotalKey, float64(stats.TotalConns), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolHitsKey, float64(stats.Hits), metrics.Labels(labels))
					m.SetGauge(ctx, metricsPoolMissesKey, float64(stats.Misses), metrics.Labels(labels))
				}
			}
		}
	})
}

func metricRedisLatency(ctx context.Context, appName, name string, labels []metrics.Label) {
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

		rdsCli := instance.GetProxy()
		begin := time.Now()
		if err := rdsCli.Ping(ctx); err != nil {
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
