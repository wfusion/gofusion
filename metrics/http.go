package metrics

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/wfusion/gofusion/common/infra/metrics/prometheus"
	"github.com/wfusion/gofusion/common/utils"

	proDrv "github.com/prometheus/client_golang/prometheus"
)

func HttpHandler(path, name string, opts ...utils.OptionExtender) http.Handler {
	opt := utils.ApplyOptions[useOption](opts...)
	rwlock.RLock()
	defer rwlock.RUnlock()

	if instances == nil || instances[opt.appName] == nil || instances[opt.appName][name] == nil {
		panic(errors.Errorf("metrics instance not found: %s %s", opt.appName, name))
	}

	// cfg := cfgsMap[opt.appName][name]
	m := utils.MapValues(instances[opt.appName][name])[0]
	switch sink := m.getProxy().(type) {
	case *prometheus.PrometheusSink:
		gatherer, ok := sink.Registry.(proDrv.Gatherer)
		if !ok {
			gatherer = proDrv.DefaultGatherer
		}
		return promhttp.InstrumentMetricHandler(
			sink.Registry, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
		)
	default:
		panic(errors.Errorf("metrics instance not support http exporter: %s %s", opt.appName, name))
	}
}
