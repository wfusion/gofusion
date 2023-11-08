package http

import (
	"context"
	"net/http"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsCodeCounterKey = []string{"http", "code", "counter"}
)

func metricsCode(ctx context.Context, appName, path, method string, code, status, rspSize int, reqSize int64) {
	select {
	case <-ctx.Done():
		return
	default:

	}

	// skip health check logging
	if path == "/health" && method == http.MethodGet {
		return
	}

	app := config.Use(appName).AppName()
	labels := []metrics.Label{
		{Key: "path", Value: path},
		{Key: "method", Value: method},
		{Key: "code", Value: cast.ToString(code)},
		{Key: "status", Value: cast.ToString(status)},
		{Key: "req_size", Value: cast.ToString(reqSize)},
		{Key: "rsp_size", Value: cast.ToString(rspSize)},
	}
	counterKey := append([]string{app}, metricsCodeCounterKey...)
	for _, m := range metrics.Internal(metrics.AppName(appName)) {
		select {
		case <-ctx.Done():
			return
		default:
			if m.IsEnableServiceLabel() {
				m.IncrCounter(ctx, counterKey, 1, metrics.Labels(labels))
			} else {
				m.IncrCounter(ctx, metricsCodeCounterKey, 1, metrics.Labels(labels))
			}
		}
	}
}
