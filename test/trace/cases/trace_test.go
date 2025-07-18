package cases

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/trace"
	"go.opentelemetry.io/otel/semconv/v1.17.0/httpconv"

	"github.com/wfusion/gofusion/log"

	testTrace "github.com/wfusion/gofusion/test/trace"
)

func TestTrace(t *testing.T) {
	testingSuite := &Trace{Test: new(testTrace.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Trace struct {
	*testTrace.Test
}

func (t *Trace) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Trace) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Trace) TestStdout() {
	t.testTrace("stdout")
}

func (t *Trace) testTrace(name string) {
	t.Run("Default", func() { t.testDefault(name) })
}

func (t *Trace) testDefault(name string) {
	t.Catch(func() {
		ctx := context.Background()
		tp := trace.Use(name, trace.AppName(t.AppName()))
		tracer := tp.Tracer("test")
		ctx, span := tracer.Start(ctx, "trace test default")
		defer func() {
			span.End()
			t.Require().True(span.SpanContext().TraceID().IsValid())
			t.Require().True(span.SpanContext().IsSampled())
			t.Require().False(span.IsRecording())
		}()

		span.SetStatus(httpconv.ServerStatus(http.StatusOK))
		t.Require().True(span.IsRecording())
	})
}
