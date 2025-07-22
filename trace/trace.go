package trace

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]TracerProvider
)

type TracerProvider interface {
	trace.TracerProvider

	config() *Conf
	shutdown(ctx context.Context) (err error)
}

type traceProvider struct {
	trace.TracerProvider
	name     string
	conf     *Conf
	exporter sdktrace.SpanExporter
}

func newTraceProvider(ctx context.Context, name string, conf *Conf,
	tp *sdktrace.TracerProvider, exporter sdktrace.SpanExporter) TracerProvider {
	return &traceProvider{name: name, conf: conf, TracerProvider: tp, exporter: exporter}
}

func (t *traceProvider) shutdown(ctx context.Context) (err error) {
	if t == nil {
		return
	}
	return t.exporter.Shutdown(ctx)
}

func (t *traceProvider) config() *Conf {
	if t == nil {
		return nil
	}
	return t.conf
}
