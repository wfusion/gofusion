package trace

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

type customSampler interface {
	trace.Sampler
	Init(ctx context.Context, conf *Conf) error
}
