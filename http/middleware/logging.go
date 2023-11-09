package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/http/consts"
	"github.com/wfusion/gofusion/log"

	fmkCtx "github.com/wfusion/gofusion/context"
)

var (
	metricsLatencyKey     = []string{"http", "latency"}
	metricsCounterKey     = []string{"http", "request", "counter"}
	metricsLatencyBuckets = []float64{
		10, 15, 20, 30, 40, 50, 60, 70, 80, 90, 99, 99.9,
		100, 150, 200, 300, 400, 500, 600, 700, 800, 900, 990, 999,
		1000, 1500, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 9900, 9990,
		10000, 15000, 20000, 30000, 40000, 50000, 60000,
	}
)

func logging(rootCtx context.Context, c *gin.Context, logger log.Logable, rawURL *url.URL, appName string) {
	ctx := fmkCtx.New(fmkCtx.Gin(c))
	cost := float64(consts.GetReqCost(c)) / float64(time.Millisecond)
	status := c.Writer.Status()
	fields := log.Fields{
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

	switch {
	case status < http.StatusBadRequest:
		logger.Info(ctx, msg, fields)
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		logger.Warn(ctx, msg, fields)
	default:
		logger.Error(ctx, msg, fields)
	}

	go metricsLogging(rootCtx, appName, rawURL.Path, c.Request.Method, status,
		c.Writer.Size(), c.Request.ContentLength, cost)
}

func metricsLogging(ctx context.Context, appName, path, method string,
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
	app := config.Use(appName).AppName()
	latencyKey := append([]string{app}, metricsLatencyKey...)
	counterKey := append([]string{app}, metricsCounterKey...)
	for _, m := range metrics.Internal(metrics.AppName(appName)) {
		select {
		case <-ctx.Done():
			return
		default:
			if m.IsEnableServiceLabel() {
				m.IncrCounter(ctx, counterKey, 1, metrics.Labels(labels))
				m.AddSample(ctx, latencyKey, cost, metrics.Labels(labels),
					metrics.PrometheusBuckets(metricsLatencyBuckets))
			} else {
				m.IncrCounter(ctx, metricsCounterKey, 1, metrics.Labels(labels))
				m.AddSample(ctx, metricsLatencyKey, cost, metrics.Labels(labels),
					metrics.PrometheusBuckets(metricsLatencyBuckets))
			}
		}
	}
}

func Logging(ctx context.Context, appName, logInstance string) gin.HandlerFunc {
	logger := log.Use(logInstance, log.AppName(appName))
	return func(c *gin.Context) {
		reqURL := clone.Clone(c.Request.URL)
		defer logging(ctx, c, logger, reqURL, appName)

		c.Next()
	}

}
