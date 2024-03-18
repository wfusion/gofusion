package http

import (
	"context"
	"net/http"

	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
)

var (
	metricsCodeTotalKey = []string{"http", "code", "total"}
)

func metricsCode(ctx context.Context, appName, path, method string, headerLabels map[string]string,
	code, status, rspSize int, reqSize int64) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	// skip health check logging
	if path == "/health" && method == http.MethodGet {
		return
	}

	_, _ = utils.Catch(func() {
		app := config.Use(appName).AppName()
		labels := []metrics.Label{
			{Key: "path", Value: path},
			{Key: "method", Value: method},
			{Key: "code", Value: cast.ToString(code)},
			{Key: "status", Value: cast.ToString(status)},
			{Key: "req_size", Value: cast.ToString(reqSize)},
			{Key: "rsp_size", Value: cast.ToString(rspSize)},
		}
		for k, v := range headerLabels {
			labels = append(labels, metrics.Label{Key: strcase.ToSnake(k), Value: v})
		}

		totalKey := append([]string{app}, metricsCodeTotalKey...)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.IncrCounter(ctx, totalKey, 1, metrics.Labels(labels))
				} else {
					m.IncrCounter(ctx, metricsCodeTotalKey, 1, metrics.Labels(labels))
				}
			}
		}
	})
}
