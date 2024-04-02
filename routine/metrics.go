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
	metricsRuntimeCpuGoroutinesKey = []string{"runtime", "cpu", "goroutines"}
	metricsRuntimeCpuCgoCallsKey   = []string{"runtime", "cpu", "cgo_calls"}
	metricsRuntimeFusGoroutinesKey = []string{"runtime", "fus", "goroutines"}

	metricsRuntimeMemAllocKey   = []string{"runtime", "mem", "alloc"}
	metricsRuntimeMemTotalKey   = []string{"runtime", "mem", "total"}
	metricsRuntimeMemSysKey     = []string{"runtime", "mem", "sys"}
	metricsRuntimeMemLookupsKey = []string{"runtime", "mem", "lookups"}
	metricsRuntimeMemMallocKey  = []string{"runtime", "mem", "malloc"}
	metricsRuntimeMemFreesKey   = []string{"runtime", "mem", "frees"}

	metricsRuntimeHeapAllocKey    = []string{"runtime", "heap", "alloc"}
	metricsRuntimeHeapSysKey      = []string{"runtime", "heap", "sys"}
	metricsRuntimeHeapIdleKey     = []string{"runtime", "heap", "idle"}
	metricsRuntimeHeapInuseKey    = []string{"runtime", "heap", "inuse"}
	metricsRuntimeHeapReleasedKey = []string{"runtime", "heap", "released"}
	metricsRuntimeHeapObjectsKey  = []string{"runtime", "heap", "objects"}

	metricsRuntimeStackInuseKey  = []string{"runtime", "stack", "inuse"}
	metricsRuntimeStackSysKey    = []string{"runtime", "stack", "sys"}
	metricsRuntimeMSpanInuseKey  = []string{"runtime", "mspan", "inuse"}
	metricsRuntimeMSpanSysKey    = []string{"runtime", "mspan", "sys"}
	metricsRuntimeMCacheInuseKey = []string{"runtime", "mcache", "inuse"}
	metricsRuntimeMCacheSysKey   = []string{"runtime", "mcache", "sys"}

	metricsRuntimeOtherSysKey = []string{"runtime", "other", "sys"}

	metricsRuntimeGCSysKey        = []string{"runtime", "gc", "sys"}
	metricsRuntimeGCNextKey       = []string{"runtime", "gc", "next"}
	metricsRuntimeGCLastKey       = []string{"runtime", "gc", "last"}
	metricsRuntimeGCCountKey      = []string{"runtime", "gc", "count"}
	metricsRuntimeGCForceCountKey = []string{"runtime", "gc", "force", "count"}
	metricsRuntimeGCPauseNSKey    = []string{"runtime", "gc", "pause_ns"}
	metricsRuntimeGCPauseTotalKey = []string{"runtime", "gc", "pause_total"}

	metricsRuntimeGCPauseNSBuckets = []float64{
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
		if idles != nil && idles[appName] != nil {
			routineNum = int64(conf.MaxRoutineAmount) - idles[appName].Load()
		}

		// export number of Goroutines
		totalRoutineNum := runtime.NumGoroutine()
		totalCgoCallsNum := runtime.NumCgoCall()

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

		cpuGoRoutinesKey := append([]string{app}, metricsRuntimeCpuGoroutinesKey...)
		cpuCgoCallsKey := append([]string{app}, metricsRuntimeCpuCgoCallsKey...)
		fusGoroutineKey := append([]string{app}, metricsRuntimeFusGoroutinesKey...)

		memAllocKey := append([]string{app}, metricsRuntimeMemAllocKey...)
		memTotalKey := append([]string{app}, metricsRuntimeMemTotalKey...)
		memSysKey := append([]string{app}, metricsRuntimeMemSysKey...)
		memLookupsKey := append([]string{app}, metricsRuntimeMemLookupsKey...)
		memMallocKey := append([]string{app}, metricsRuntimeMemMallocKey...)
		memFreesKey := append([]string{app}, metricsRuntimeMemFreesKey...)

		heapAllocKey := append([]string{app}, metricsRuntimeHeapAllocKey...)
		heapSysKey := append([]string{app}, metricsRuntimeHeapSysKey...)
		heapIdleKey := append([]string{app}, metricsRuntimeHeapIdleKey...)
		heapInuseKey := append([]string{app}, metricsRuntimeHeapInuseKey...)
		heapReleasedKey := append([]string{app}, metricsRuntimeHeapReleasedKey...)
		heapObjectsKey := append([]string{app}, metricsRuntimeHeapObjectsKey...)

		stackInuseKey := append([]string{app}, metricsRuntimeStackInuseKey...)
		stackSysKey := append([]string{app}, metricsRuntimeStackSysKey...)
		mspanInuseKey := append([]string{app}, metricsRuntimeMSpanInuseKey...)
		mspanSysKey := append([]string{app}, metricsRuntimeMSpanSysKey...)
		mcacheInuseKey := append([]string{app}, metricsRuntimeMCacheInuseKey...)
		mcacheSysKey := append([]string{app}, metricsRuntimeMCacheSysKey...)

		otherSysKey := append([]string{app}, metricsRuntimeOtherSysKey...)

		gcSysKey := append([]string{app}, metricsRuntimeGCSysKey...)
		gcNextKey := append([]string{app}, metricsRuntimeGCNextKey...)
		gcLastKey := append([]string{app}, metricsRuntimeGCLastKey...)
		gcCountKey := append([]string{app}, metricsRuntimeGCCountKey...)
		gcForceCountKey := append([]string{app}, metricsRuntimeGCForceCountKey...)
		gcPauseNSKey := append([]string{app}, metricsRuntimeGCPauseNSKey...)
		gcPauseTotalKey := append([]string{app}, metricsRuntimeGCPauseTotalKey...)

		metricsLabels := metrics.Labels(labels)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.SetGauge(ctx, cpuGoRoutinesKey, float64(totalRoutineNum), metricsLabels)
					m.SetGauge(ctx, cpuCgoCallsKey, float64(totalCgoCallsNum), metricsLabels)
					m.SetGauge(ctx, fusGoroutineKey, float64(routineNum), metricsLabels)

					m.SetGauge(ctx, memAllocKey, float64(stats.Alloc), metricsLabels)
					m.SetGauge(ctx, memTotalKey, float64(stats.TotalAlloc), metricsLabels)
					m.SetGauge(ctx, memSysKey, float64(stats.Sys), metricsLabels)
					m.SetGauge(ctx, memLookupsKey, float64(stats.Lookups), metricsLabels)
					m.SetGauge(ctx, memMallocKey, float64(stats.Mallocs), metricsLabels)
					m.SetGauge(ctx, memFreesKey, float64(stats.Frees), metricsLabels)

					m.SetGauge(ctx, heapAllocKey, float64(stats.HeapAlloc), metricsLabels)
					m.SetGauge(ctx, heapSysKey, float64(stats.HeapSys), metricsLabels)
					m.SetGauge(ctx, heapIdleKey, float64(stats.HeapIdle), metricsLabels)
					m.SetGauge(ctx, heapInuseKey, float64(stats.HeapInuse), metricsLabels)
					m.SetGauge(ctx, heapReleasedKey, float64(stats.HeapReleased), metricsLabels)
					m.SetGauge(ctx, heapObjectsKey, float64(stats.HeapObjects), metricsLabels)

					m.SetGauge(ctx, stackInuseKey, float64(stats.StackInuse), metricsLabels)
					m.SetGauge(ctx, stackSysKey, float64(stats.StackSys), metricsLabels)
					m.SetGauge(ctx, mspanInuseKey, float64(stats.MSpanInuse), metricsLabels)
					m.SetGauge(ctx, mspanSysKey, float64(stats.MSpanSys), metricsLabels)
					m.SetGauge(ctx, mcacheInuseKey, float64(stats.MCacheInuse), metricsLabels)
					m.SetGauge(ctx, mcacheSysKey, float64(stats.MCacheSys), metricsLabels)

					m.SetGauge(ctx, otherSysKey, float64(stats.OtherSys), metricsLabels)

					m.SetGauge(ctx, gcSysKey, float64(stats.GCSys), metricsLabels)
					m.SetGauge(ctx, gcNextKey, float64(stats.NextGC), metricsLabels)
					m.SetGauge(ctx, gcLastKey, float64(stats.LastGC), metricsLabels)
					m.SetGauge(ctx, gcCountKey, float64(stats.NumGC), metricsLabels)
					m.SetGauge(ctx, gcForceCountKey, float64(stats.NumForcedGC), metricsLabels)
					m.SetGauge(ctx, gcPauseTotalKey, float64(stats.PauseTotalNs), metricsLabels)
					for i := lastNumGc.Load(); i < stats.NumGC; i++ {
						m.AddSample(ctx, gcPauseNSKey, float64(stats.PauseNs[i%256]),
							metricsLabels, metrics.PrometheusBuckets(metricsRuntimeGCPauseNSBuckets))
					}
				} else {
					m.SetGauge(ctx, metricsRuntimeCpuGoroutinesKey, float64(totalRoutineNum), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeCpuCgoCallsKey, float64(totalCgoCallsNum), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeFusGoroutinesKey, float64(routineNum), metricsLabels)

					m.SetGauge(ctx, metricsRuntimeMemAllocKey, float64(stats.Alloc), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMemTotalKey, float64(stats.TotalAlloc), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMemSysKey, float64(stats.Sys), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMemLookupsKey, float64(stats.Lookups), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMemMallocKey, float64(stats.Mallocs), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMemFreesKey, float64(stats.Frees), metricsLabels)

					m.SetGauge(ctx, metricsRuntimeHeapAllocKey, float64(stats.HeapAlloc), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeHeapSysKey, float64(stats.HeapSys), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeHeapIdleKey, float64(stats.HeapIdle), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeHeapInuseKey, float64(stats.HeapInuse), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeHeapReleasedKey, float64(stats.HeapReleased), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeHeapObjectsKey, float64(stats.HeapObjects), metricsLabels)

					m.SetGauge(ctx, metricsRuntimeStackInuseKey, float64(stats.StackInuse), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeStackSysKey, float64(stats.StackSys), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMSpanInuseKey, float64(stats.MSpanInuse), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMSpanSysKey, float64(stats.MSpanSys), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMCacheInuseKey, float64(stats.MCacheInuse), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeMCacheSysKey, float64(stats.MCacheSys), metricsLabels)

					m.SetGauge(ctx, metricsRuntimeOtherSysKey, float64(stats.OtherSys), metricsLabels)

					m.SetGauge(ctx, metricsRuntimeGCSysKey, float64(stats.GCSys), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeGCNextKey, float64(stats.NextGC), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeGCLastKey, float64(stats.LastGC), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeGCCountKey, float64(stats.NumGC), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeGCForceCountKey, float64(stats.NumForcedGC), metricsLabels)
					m.SetGauge(ctx, metricsRuntimeGCPauseTotalKey, float64(stats.PauseTotalNs), metricsLabels)
					for i := lastNumGc.Load(); i < stats.NumGC; i++ {
						m.AddSample(ctx, metricsRuntimeGCPauseNSKey, float64(stats.PauseNs[i%256]),
							metricsLabels,
							metrics.PrometheusBuckets(metricsRuntimeGCPauseNSBuckets),
						)
					}
				}
			}
		}

		lastNumGc.Store(stats.NumGC)
	})
}
