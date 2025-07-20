package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.17.0/httpconv"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/metrics"
	"github.com/wfusion/gofusion/trace"

	fusCtx "github.com/wfusion/gofusion/context"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	otlTra "go.opentelemetry.io/otel/trace"
)

var (
	metricsStatusTotalKey = []string{"http", "status", "total"}
)

func Trace(ctx context.Context, appName string, traceHeaders, userHeaders, headerLabels []string) gin.HandlerFunc {
	propagator := otel.GetTextMapPropagator()
	return func(c *gin.Context) {
		defer func() { go metricsStatus(ctx, c, appName, headerLabels) }()

		extractTraceAndUser(c, traceHeaders, userHeaders)
		traceStatus(ctx, c, appName, propagator, c.Next)
	}
}

func extractTraceAndUser(c *gin.Context, traceHeaders, userHeaders []string) {
	var (
		userID, traceID string
	)
	utils.IfAny(
		func() bool { traceID = c.GetHeader(fusCtx.KeyTraceID); return traceID != "" },
		func() bool {
			for _, header := range traceHeaders {
				if traceID = c.GetHeader(header); traceID != "" {
					return true
				}
			}
			return false
		},
		func() bool { traceID = c.GetHeader("traceid"); return traceID != "" },
		func() bool { traceID = c.GetHeader("HTTP_TRACE_ID"); return traceID != "" },
		func() bool {
			traceID = utils.LookupByFuzzyKeyword[string](c.GetHeader, "trace_id")
			return traceID != ""
		},
		func() bool { traceID = utils.NginxID(); return traceID != "" },
	)
	c.Header("traceid", traceID)
	c.Set(fusCtx.KeyTraceID, traceID)

	utils.IfAny(
		func() bool { userID = c.GetHeader(fusCtx.KeyUserID); return userID != "" },
		func() bool {
			for _, header := range userHeaders {
				if traceID = c.GetHeader(header); traceID != "" {
					return true
				}
			}
			return false
		},
		func() bool {
			userID = utils.LookupByFuzzyKeyword[string](c.GetHeader, "user_id")
			return userID != ""
		},
		func() bool {
			userID = utils.LookupByFuzzyKeyword[string](c.GetQuery, "user_id")
			return userID != ""
		},
		func() bool {
			userID = utils.LookupByFuzzyKeyword[string](c.GetPostForm, "user_id")
			return userID != ""
		},
	)
	c.Header("userid", userID)
	c.Set(fusCtx.KeyUserID, userID)
}

func metricsStatus(ctx context.Context, c *gin.Context, appName string, headerLabels []string) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	path := c.Request.URL.Path
	method := c.Request.Method
	status := c.Writer.Status()
	reqSize := c.Request.ContentLength
	rspSize := c.Writer.Size()

	// skip health check logging
	if path == "/health" && method == http.MethodGet {
		return
	}

	_, _ = utils.Catch(func() {
		app := config.Use(appName).AppName()
		labels := []metrics.Label{
			{Key: "path", Value: path},
			{Key: "method", Value: method},
			{Key: "status", Value: cast.ToString(status)},
			{Key: "req_size", Value: cast.ToString(reqSize)},
			{Key: "rsp_size", Value: cast.ToString(rspSize)},
		}
		for k, v := range parseHeaderMetrics(c, headerLabels) {
			labels = append(labels, metrics.Label{Key: strcase.ToSnake(k), Value: v})
		}

		totalKey := append([]string{app}, metricsStatusTotalKey...)
		for _, m := range metrics.Internal(metrics.AppName(appName)) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.IsEnableServiceLabel() {
					m.IncrCounter(ctx, totalKey, 1, metrics.Labels(labels))
				} else {
					m.IncrCounter(ctx, metricsStatusTotalKey, 1, metrics.Labels(labels))
				}
			}
		}
	})
}

func traceStatus(ctx context.Context, c *gin.Context, appName string,
	propagator propagation.TextMapPropagator, next func()) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	_, _ = utils.Catch(func() {
		app := config.Use(appName).AppName()
		fullPath := c.FullPath()
		spanName := fmt.Sprintf("%v %v", c.Request.Method, fullPath)

		traceCtx := c.Request.Context()
		if propagator != nil {
			traceCtx = propagator.Extract(traceCtx, propagation.HeaderCarrier(c.Request.Header))
		}
		opts := []otlTra.SpanStartOption{
			otlTra.WithSpanKind(otlTra.SpanKindServer),
			otlTra.WithAttributes(semconv.HTTPRoute(fullPath)),
		}

		tps := trace.Internal(trace.AppName(app))
		spanList := make([]otlTra.Span, 0, len(tps))
		for _, tp := range tps {
			select {
			case <-ctx.Done():
				return
			default:
				spanCtx, span := tp.Tracer("").Start(traceCtx, spanName, opts...)
				c.Request = c.Request.WithContext(spanCtx)
				spanList = append(spanList, span)
			}
		}

		defer func() {
			for _, span := range spanList {
				select {
				case <-ctx.Done():
					return
				default:
				}

				status := c.Writer.Status()
				span.SetStatus(httpconv.ServerStatus(status))
				if status > 0 {
					span.SetAttributes(semconv.HTTPStatusCode(status))
				}
				if len(c.Errors) > 0 {
					span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
				}
			}
			for _, span := range spanList {
				select {
				case <-ctx.Done():
					return
				default:
				}

				span.End()
			}
		}()
		next()
	})
}

func parseHeaderMetrics(c *gin.Context, labels []string) (headerLabels map[string]string) {
	headerLabels = make(map[string]string, len(labels))
	for _, metricsHeader := range labels {
		headerLabels[metricsHeader] = c.Request.Header.Get(metricsHeader)
	}
	return
}
