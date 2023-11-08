package metrics

import (
	"context"

	"github.com/wfusion/gofusion/common/infra/metrics"
)

type mock struct {
	*abstract
}

func newMock(ctx context.Context, appName, name, job string, conf *cfg) Sink {
	sink := new(metrics.BlackholeSink)
	return &mock{abstract: newMetrics(ctx, appName, name, job, sink, conf)}
}
