package kv

import (
	"context"
	"log"
	"sync"
	"syscall"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/spf13/cast"
	"go.etcd.io/etcd/client/v3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
	"github.com/wfusion/gofusion/redis"
)

var (
	metricsLatencyKey = []string{"kv", "latency"}

	// redis
	metricsPoolIdleKey   = []string{"kv", "pool", "idle"}
	metricsPoolTotalKey  = []string{"kv", "pool", "total"}
	metricsPoolStaleKey  = []string{"kv", "pool", "stale"}
	metricsPoolHitsKey   = []string{"kv", "pool", "hits"}
	metricsPoolMissesKey = []string{"kv", "pool", "misses"}

	// etcd
	metricsRaftIndexKey        = []string{"kv", "raft", "index"}
	metricsRaftTermKey         = []string{"kv", "raft", "term"}
	metricsRaftAppliedIndexKey = []string{"kv", "raft", "applied", "index"}
	metricsDBSizeKey           = []string{"kv", "db", "size"}
	metricsDBSizeInUseKey      = []string{"kv", "db", "size", "in", "use"}
	metricsLatencyBuckets      = []float64{
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
	}

	if conf.Endpoint.RedisDB > 0 {
		labels = append(labels, metrics.Label{Key: "database", Value: cast.ToString(conf.Endpoint.RedisDB)})
	}

	log.Printf("%v [Gofusion] %s %s %s metrics start", syscall.Getpid(), app, config.ComponentKV, name)
	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentRedis, name)
			return
		case <-ticker:
			switch conf.Type {
			case kvTypeRedis:
				go metricRedisStats(ctx, appName, name, labels)
				go metricRedisLatency(ctx, appName, name, labels)
			case kvTypeConsul:
				// TODO: emit each http request latency?
				log.Printf("%v [Gofusion] %s %s %s metrics exited cause nothing to emit",
					syscall.Getpid(), app, config.ComponentKV, name)
				return
			case kvTypeEtcd:
				go metricsEtcd(ctx, appName, name, labels)
			case kvTypeZK:
				go metricsZK(ctx, appName, name, labels)
			}
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
		instances, ok := appInstances[appName]
		if !ok {
			return
		}
		instance, ok := instances[name]
		if !ok {
			return
		}

		app := config.Use(appName).AppName()
		idleKey := append([]string{app}, metricsPoolIdleKey...)
		staleKey := append([]string{app}, metricsPoolStaleKey...)
		totalKey := append([]string{app}, metricsPoolTotalKey...)
		hitsKey := append([]string{app}, metricsPoolHitsKey...)
		missesKey := append([]string{app}, metricsPoolMissesKey...)

		cli := instance.getProxy().(*redis.Redis)
		stats := cli.PoolStats()
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

		cli := instance.getProxy().(*redis.Redis)
		begin := time.Now()
		if err := cli.Ping(ctx); err != nil {
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

func metricsEtcd(ctx context.Context, appName, name string, labels []metrics.Label) {
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

		app := config.Use(appName).AppName()
		latencyKey := append([]string{app}, metricsLatencyKey...)
		dbSizeKey := append([]string{app}, metricsDBSizeKey...)
		dbSizeInUseKey := append([]string{app}, metricsDBSizeInUseKey...)
		raftIndexKey := append([]string{app}, metricsRaftIndexKey...)
		raftTermKey := append([]string{app}, metricsRaftTermKey...)
		raftAppliedIndexKey := append([]string{app}, metricsRaftAppliedIndexKey...)

		cli := instance.getProxy().(*clientv3.Client)
		etcdLabels := make([]metrics.Label, len(labels))
		copy(etcdLabels, labels)

		activeConnStat := cli.ActiveConnection()
		etcdLabels = append(etcdLabels, metrics.Label{Key: "conn_target", Value: activeConnStat.Target()})
		etcdLabels = append(etcdLabels, metrics.Label{Key: "conn_state", Value: activeConnStat.GetState().String()})

		wg := new(sync.WaitGroup)
		for _, addr := range cli.Endpoints() {
			wg.Add(1)
			go func(addr string) {
				_, _ = utils.Catch(func() {
					defer wg.Done()

					begin := time.Now()
					rsp, err := cli.Status(ctx, addr)
					if err != nil {
						log.Printf("%v [Gofusion] %s %s %s call etcd status api failed: %s",
							syscall.Getpid(), appName, config.ComponentKV, name, err)
						return
					}
					latency := float64(time.Since(begin)) / float64(time.Millisecond)
					for _, m := range metrics.Internal(metrics.AppName(appName)) {
						select {
						case <-ctx.Done():
							return
						default:
							epLabels := make([]metrics.Label, len(etcdLabels))
							copy(epLabels, etcdLabels)
							epLabels = append(epLabels,
								metrics.Label{Key: "endpoint", Value: addr},
								metrics.Label{Key: "raft_learner", Value: cast.ToString(rsp.IsLearner)},
							)
							labelOption := metrics.Labels(epLabels)

							if m.IsEnableServiceLabel() {
								m.AddSample(ctx, latencyKey, latency,
									labelOption,
									metrics.PrometheusBuckets(metricsLatencyBuckets),
								)
								m.SetGauge(ctx, dbSizeKey, float64(rsp.DbSize), labelOption)
								m.SetGauge(ctx, dbSizeInUseKey, float64(rsp.DbSizeInUse), labelOption)
								m.SetGauge(ctx, raftIndexKey, float64(rsp.RaftIndex), labelOption)
								m.SetGauge(ctx, raftTermKey, float64(rsp.RaftTerm), labelOption)
								m.SetGauge(ctx, raftAppliedIndexKey, float64(rsp.RaftAppliedIndex), labelOption)

							} else {
								m.AddSample(ctx, metricsLatencyKey, latency,
									metrics.Labels(epLabels),
									metrics.PrometheusBuckets(metricsLatencyBuckets),
								)
								m.SetGauge(ctx, metricsDBSizeKey, float64(rsp.DbSize), labelOption)
								m.SetGauge(ctx, metricsDBSizeInUseKey, float64(rsp.DbSizeInUse), labelOption)
								m.SetGauge(ctx, metricsRaftIndexKey, float64(rsp.RaftIndex), labelOption)
								m.SetGauge(ctx, metricsRaftTermKey, float64(rsp.RaftTerm), labelOption)
								m.SetGauge(ctx, metricsRaftAppliedIndexKey, float64(rsp.RaftAppliedIndex),
									metrics.Labels(epLabels))
							}
						}
					}
				})
			}(addr)
		}
		wg.Wait()
	})
}

func metricsZK(ctx context.Context, appName, name string, labels []metrics.Label) {
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

		app := config.Use(appName).AppName()
		latencyKey := append([]string{app}, metricsLatencyKey...)

		cli := instance.getProxy().(*zk.Conn)
		zkLabels := make([]metrics.Label, len(labels))
		copy(zkLabels, labels)

		zkLabels = append(zkLabels, metrics.Label{Key: "conn_state", Value: cli.State().String()})
		zkLabels = append(zkLabels, metrics.Label{Key: "session_id", Value: cast.ToString(cli.SessionID())})
		zkLabels = append(zkLabels, metrics.Label{Key: "server", Value: cast.ToString(cli.Server())})

		begin := time.Now()
		_, _, err := cli.Exists("/")
		if err != nil {
			log.Printf("%v [Gofusion] %s %s %s call zookeeper root exists failed: %s",
				syscall.Getpid(), appName, config.ComponentKV, name, err)
			return
		}
		latency := float64(time.Since(begin)) / float64(time.Millisecond)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.AddSample(ctx, latencyKey, latency,
						metrics.Labels(zkLabels),
						metrics.PrometheusBuckets(metricsLatencyBuckets),
					)
				} else {
					m.AddSample(ctx, metricsLatencyKey, latency,
						metrics.Labels(zkLabels),
						metrics.PrometheusBuckets(metricsLatencyBuckets),
					)
				}
			}
		}
	})
}
