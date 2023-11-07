package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/metrics/prometheus"

	proDrv "github.com/prometheus/client_golang/prometheus"
)

var (
	prometheusRWLocker  = new(sync.RWMutex)
	prometheusRegisters = map[string]proDrv.Registerer{}
)

type _Prometheus struct {
	*abstract
}

func newPrometheusPush(ctx context.Context, appName, name, job string, interval time.Duration, conf *cfg) Sink {
	sink, err := prometheus.NewPrometheusPushSink(ctx, conf.c.Endpoint.Addresses[0], interval, job, conf.logger)
	if err != nil {
		panic(errors.Errorf("initialize metrics component push mode prometheus failed: %s", err))
	}
	return &_Prometheus{abstract: newMetrics(ctx, appName, name, job, sink, conf)}
}

func newPrometheusPull(ctx context.Context, appName, name, job string, conf *cfg) Sink {
	prometheusRWLocker.Lock()
	if _, ok := prometheusRegisters[appName]; !ok {
		prometheusRegisters[appName] = proDrv.NewRegistry()
	}
	prometheusRWLocker.Unlock()

	sink, err := prometheus.NewPrometheusSinkFrom(prometheus.PrometheusOpts{
		Expiration:           prometheus.DefaultPrometheusOpts.Expiration,
		Registerer:           prometheusRegisters[appName],
		GaugeDefinitions:     nil,
		SummaryDefinitions:   nil,
		CounterDefinitions:   nil,
		HistogramDefinitions: nil,
		Name:                 job,
	})
	if err != nil {
		panic(errors.Errorf("initialize metrics component pull mode prometheus failed: %s", err))
	}

	return &_Prometheus{abstract: newMetrics(ctx, appName, name, job, sink, conf)}
}
