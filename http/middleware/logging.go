package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/http/consts"

	fusCtx "github.com/wfusion/gofusion/context"
	fusLog "github.com/wfusion/gofusion/log"
)

var (
	metricsLatencyKey     = []string{"http", "latency"}
	metricsTotalKey       = []string{"http", "request", "total"}
	metricsLatencyBuckets = []float64{
		10, 15, 20, 30, 40, 50, 60, 70, 80, 90, 99, 99.9,
		100, 150, 200, 300, 400, 500, 600, 700, 800, 900, 990, 999,
		1000, 1500, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 9900, 9990,
		10000, 15000, 20000, 30000, 40000, 50000, 60000,
	}
)

func logging(rootCtx context.Context, c *gin.Context, logger resty.Logger, rawURL *url.URL, metricsHeaders []string,
	appName string) {
	ctx := fusCtx.New(fusCtx.Gin(c))
	cost := float64(consts.GetReqCost(c)) / float64(time.Millisecond)
	status := c.Writer.Status()
	fields := fusLog.Fields{
		"path":        rawURL.Path,
		"method":      c.Request.Method,
		"status":      status,
		"referer":     c.Request.Referer(),
		"req_size":    c.Request.ContentLength,
		"rsp_size":    c.Writer.Size(),
		"cost":        cost,
		"user_agent":  c.Request.UserAgent(),
		"client_addr": c.ClientIP(),
	}

	// skip health check logging
	if rawURL.Path == "/health" && c.Request.Method == http.MethodGet {
		return
	}

	msg := fmt.Sprintf(
		"%s -> %s %s %d %d %d (%.2fms)",
		c.ClientIP(), utils.LocalIP.String(),
		strconv.Quote(fmt.Sprintf("%s %s", c.Request.Method, rawURL)),
		c.Request.ContentLength, c.Writer.Size(), status, cost,
	)

	if logger != nil {
		switch {
		case status < http.StatusBadRequest:
			logger.Debugf(msg, ctx, fields)
		case status < http.StatusInternalServerError:
			logger.Warnf(msg, ctx, fields)
		default:
			logger.Errorf(msg, ctx, fields)
		}
	} else {
		log.Printf(msg+" %s", utils.MustJsonMarshal(fields))
	}

	headerLabels := make(map[string]string, len(metricsHeaders))
	for _, metricsHeader := range metricsHeaders {
		headerLabels[metricsHeader] = c.Request.Header.Get(metricsHeader)
	}

	go metricsLogging(rootCtx, appName, rawURL.Path, c.Request.Method, headerLabels,
		status, c.Writer.Size(), c.Request.ContentLength, cost)
}

func metricsLogging(ctx context.Context, appName, path, method string, headerLabels map[string]string,
	status, rspSize int, reqSize int64, cost float64) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	labels := []metrics.Label{
		{Key: "path", Value: path},
		{Key: "method", Value: method},
		{Key: "status", Value: cast.ToString(status)},
		{Key: "req_size", Value: cast.ToString(reqSize)},
		{Key: "rsp_size", Value: cast.ToString(rspSize)},
	}
	for k, v := range headerLabels {
		labels = append(labels, metrics.Label{Key: strcase.ToSnake(k), Value: v})
	}

	app := config.Use(appName).AppName()
	latencyKey := append([]string{app}, metricsLatencyKey...)
	totalKey := append([]string{app}, metricsTotalKey...)
	for _, m := range metrics.Internal(metrics.AppName(appName)) {
		select {
		case <-ctx.Done():
			return
		default:
			if m.IsEnableServiceLabel() {
				m.IncrCounter(ctx, totalKey, 1, metrics.Labels(labels))
				m.AddSample(ctx, latencyKey, cost, metrics.Labels(labels),
					metrics.PrometheusBuckets(metricsLatencyBuckets))
			} else {
				m.IncrCounter(ctx, metricsTotalKey, 1, metrics.Labels(labels))
				m.AddSample(ctx, metricsLatencyKey, cost, metrics.Labels(labels),
					metrics.PrometheusBuckets(metricsLatencyBuckets))
			}
		}
	}
}

func Logging(ctx context.Context, appName string, metricsHeaders []string, logger resty.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqURL := clone.Clone(c.Request.URL)
		defer logging(ctx, c, logger, reqURL, metricsHeaders, appName)

		c.Next()
	}

}
