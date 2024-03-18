package routine

import (
	"context"
	"log"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsRuntimeTotalGoroutinesKey     = []string{"runtime", "total_goroutines"}
	metricsRuntimeGoroutinesKey          = []string{"runtime", "fus_goroutines"}
	metricsRuntimeAllocBytesKey          = []string{"runtime", "alloc_bytes"}
	metricsRuntimeSysBytesKey            = []string{"runtime", "sys_bytes"}
	metricsRuntimeMallocCountKey         = []string{"runtime", "malloc_count"}
	metricsRuntimeFreeCountKey           = []string{"runtime", "free_count"}
	metricsRuntimeHeapObjectsKey         = []string{"runtime", "heap_objects"}
	metricsRuntimeGCRunsKey              = []string{"runtime", "total_gc_runs"}
	metricsRuntimeTotalSTWLatencyKey     = []string{"runtime", "total_gc_pause_ns"}
	metricsRuntimeTotalSTWLatencyBuckets = []float64{
		1000, 2500, 5000, 7500, 9000, 9500, 9900, // 1μs - 9.9μs
		10000, 25000, 50000, 75000, 90000, 95000, 99000, // 0.01ms - 0.099ms
		100000, 250000, 500000, 750000, 900000, 950000, 990000, // 0.1ms - 0.99ms
		1000000, 2500000, 5000000, 7500000, 9000000, 9500000, 9900000, // 1ms - 9.9ms
		10000000, 25000000, 50000000, 75000000, 90000000, 95000000, 99000000, // 10ms - 99ms
		100000000, 250000000, 500000000, 750000000, 900000000, 950000000, 990000000, // 100ms - 990ms
		1000000000, 2500000000, 5000000000, 7500000000, 9000000000, 9500000000, 9900000000, // 1s - 9.9s
		10000000000, 25000000000, 50000000000, 75000000000, 90000000000, 95000000000, 99000000000, // 10s - 99s
	}
)

func startDaemonRoutines(ctx context.Context, appName string, conf *Conf) {
	ticker := time.Tick(time.Second * 5)
	app := config.Use(appName).AppName()
	labels := make([]metrics.Label, 0)
	lastNumGc := atomic.NewUint32(0)

	log.Printf("%v [Gofusion] %s %s metrics start", syscall.Getpid(), app, config.ComponentGoroutinePool)
	for {
		select {
		case <-ctx.Done():
			log.Printf("%v [Gofusion] %s %s metrics exited",
				syscall.Getpid(), app, config.ComponentGoroutinePool)
			return
		case <-ticker:
			go metricsRuntime(ctx, appName, lastNumGc, conf, labels)
		}
	}
}

func metricsRuntime(ctx context.Context, appName string, lastNumGc *atomic.Uint32, conf *Conf, labels []metrics.Label) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	_, _ = utils.Catch(func() {
		var routineNum int64
		if idle != nil && idle[appName] != nil {
			routineNum = int64(conf.MaxRoutineAmount) - idle[appName].Load()
		}

		// export number of Goroutines
		totalRoutineNum := runtime.NumGoroutine()

		// export memory stats
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		// export info about the last few GC runs
		// handle wrap around
		if stats.NumGC < lastNumGc.Load() {
			lastNumGc.Store(0)
		}

		// ensure we don't scan more than 256
		if stats.NumGC-lastNumGc.Load() >= 256 {
			lastNumGc.Store(stats.NumGC - 255)
		}

		app := config.Use(appName).AppName()
		totalGoRoutinesKey := append([]string{app}, metricsRuntimeTotalGoroutinesKey...)
		goroutineKey := append([]string{app}, metricsRuntimeGoroutinesKey...)
		allocBytesKey := append([]string{app}, metricsRuntimeAllocBytesKey...)
		sysBytesKey := append([]string{app}, metricsRuntimeSysBytesKey...)
		mallocCountKey := append([]string{app}, metricsRuntimeMallocCountKey...)
		freeCountKey := append([]string{app}, metricsRuntimeFreeCountKey...)
		heapObjectsKey := append([]string{app}, metricsRuntimeHeapObjectsKey...)
		gcRunsKey := append([]string{app}, metricsRuntimeGCRunsKey...)
		totalSTWLatencyKey := append([]string{app}, metricsRuntimeTotalSTWLatencyKey...)

		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.SetGauge(ctx, totalGoRoutinesKey, float64(totalRoutineNum), metrics.Labels(labels))
					m.SetGauge(ctx, goroutineKey, float64(routineNum), metrics.Labels(labels))
					m.SetGauge(ctx, allocBytesKey, float64(stats.Alloc), metrics.Labels(labels))
					m.SetGauge(ctx, sysBytesKey, float64(stats.Sys), metrics.Labels(labels))
					m.SetGauge(ctx, mallocCountKey, float64(stats.Mallocs), metrics.Labels(labels))
					m.SetGauge(ctx, freeCountKey, float64(stats.Frees), metrics.Labels(labels))
					m.SetGauge(ctx, heapObjectsKey, float64(stats.HeapObjects), metrics.Labels(labels))
					m.SetGauge(ctx, gcRunsKey, float64(stats.NumGC), metrics.Labels(labels))
					for i := lastNumGc.Load(); i < stats.NumGC; i++ {
						m.AddSample(ctx, totalSTWLatencyKey, float64(stats.PauseNs[i%256]), metrics.Labels(labels))
					}
				} else {
					m.SetGauge(ctx, metricsRuntimeTotalGoroutinesKey, float64(totalRoutineNum), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeGoroutinesKey, float64(routineNum), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeAllocBytesKey, float64(stats.Alloc), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeSysBytesKey, float64(stats.Sys), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeMallocCountKey, float64(stats.Mallocs), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeFreeCountKey, float64(stats.Frees), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeHeapObjectsKey, float64(stats.HeapObjects), metrics.Labels(labels))
					m.SetGauge(ctx, metricsRuntimeGCRunsKey, float64(stats.NumGC), metrics.Labels(labels))
					for i := lastNumGc.Load(); i < stats.NumGC; i++ {
						m.AddSample(ctx, metricsRuntimeTotalSTWLatencyKey, float64(stats.PauseNs[i%256]),
							metrics.Labels(labels),
							metrics.PrometheusBuckets(metricsRuntimeTotalSTWLatencyBuckets),
						)
					}
				}
			}
		}

		lastNumGc.Store(stats.NumGC)
	})
}
