package trace

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]TracerProvider
)

type TracerProvider interface {
	trace.TracerProvider

	config() *Conf
}

type traceProvider struct {
	trace.TracerProvider
	ctx  context.Context
	name string
	conf *Conf
}

func newTraceProvider(ctx context.Context, name string, conf *Conf, tp trace.TracerProvider) TracerProvider {
	return &traceProvider{ctx: ctx, name: name, conf: conf, TracerProvider: tp}
}

func (t *traceProvider) config() *Conf {
	if t == nil {
		return nil
	}
	return t.conf
}
