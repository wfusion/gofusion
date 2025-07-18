package trace

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

type customPropagator interface {
	propagation.TextMapPropagator
	Init(ctx context.Context, conf *Conf) error
}
