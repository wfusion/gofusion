package trace

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

type customIDGenerator interface {
	trace.IDGenerator
	Init(ctx context.Context, conf *Conf) error
}
