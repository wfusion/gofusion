package cases

import (
	"context"
	"math"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/metrics"
	"github.com/wfusion/gofusion/test/internal/mock"

	testMetrics "github.com/wfusion/gofusion/test/metrics"
)

func TestPush(t *testing.T) {
	testingSuite := &Push{Test: new(testMetrics.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Push struct {
	*testMetrics.Test
}

func (t *Push) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Push) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Push) TestPrometheus() {
	t.testDefault(namePrometheusPush)
}

func (t *Push) testDefault(name string) {
	t.Run("testPush", func() { t.testPush(name) })
	t.Run("testConcurrency", func() { t.testConcurrency(name) })
}

func (t *Push) testPush(name string) {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		labels := []metrics.Label{
			{
				Key:   "hostname",
				Value: utils.Must(os.Hostname()),
			},
			{
				Key:   "ip",
				Value: utils.ClientIP(),
			},
		}

		job := name + "TestPush"
		sink := metrics.Use(name, job, metrics.AppName(t.AppName()))

		// When
		sink.SetGauge(ctx, []string{"gauge", "without", "labels"}, mock.GenObj[float64]())
		sink.SetGauge(ctx, []string{"gauge", "with", "labels"}, mock.GenObj[float64](),
			metrics.Labels(labels))
		sink.SetGauge(ctx, []string{"gauge", "with", "buckets"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)))
		sink.SetGauge(ctx, []string{"gauge", "with", "precision"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.Precision())

		sink.IncrCounter(ctx, []string{"counter", "without", "labels"}, float64(mock.GenObj[uint8]()))
		sink.IncrCounter(ctx, []string{"counter", "with", "labels"}, float64(mock.GenObj[uint8]()),
			metrics.Labels(labels))

		sink.AddSample(ctx, []string{"sample", "without", "labels"}, mock.GenObj[float64]())
		sink.AddSample(ctx, []string{"sample", "with", "labels"}, mock.GenObj[float64](),
			metrics.Labels(labels))
		sink.AddSample(ctx, []string{"sample", "with", "buckets"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)))
		sink.AddSample(ctx, []string{"sample", "with", "precision"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.Precision())

		sink.MeasureSince(ctx, []string{"measure", "without", "labels"}, mock.GenObj[time.Time]())
		sink.MeasureSince(ctx, []string{"measure", "with", "labels"}, mock.GenObj[time.Time](),
			metrics.Labels(labels))
		sink.MeasureSince(ctx, []string{"measure", "with", "buckets"}, mock.GenObj[time.Time](),
			metrics.Labels(labels),
			metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)))
		sink.MeasureSince(ctx, []string{"measure", "with", "precision"}, mock.GenObj[time.Time](),
			metrics.Labels(labels),
			metrics.Precision())

		sink.AddSample(ctx, []string{"sample", "with", "timeout"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.Precision(), metrics.Timeout(100*time.Millisecond))

		sink.AddSample(ctx, []string{"sample", "with", "must"}, mock.GenObj[float64](),
			metrics.Labels(labels),
			metrics.Precision(), metrics.WithoutTimeout())

		// Then
		time.Sleep(timeout)
	})
}

func (t *Push) testConcurrency(name string) {
	t.Catch(func() {
		// Given
		ctx := context.Background()

		labels := []metrics.Label{
			{
				Key:   "hostname",
				Value: utils.Must(os.Hostname()),
			},
			{
				Key:   "ip",
				Value: utils.ClientIP(),
			},
		}

		job := name + "TestConcurrency"
		sink := metrics.Use(name, job, metrics.AppName(t.AppName()))

		// When
		wg := new(sync.WaitGroup)
		round := 1000
		concurrency := runtime.NumCPU()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < round; i++ {
					sink.SetGauge(ctx, []string{"gauge", "without", "labels"}, mock.GenObj[float64](),
						metrics.WithoutTimeout())

					sink.SetGauge(ctx, []string{"gauge", "with", "labels"},
						mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.WithoutTimeout())

					sink.SetGauge(ctx, []string{"gauge", "with", "buckets"}, mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)),
						metrics.WithoutTimeout())

					sink.SetGauge(ctx, []string{"gauge", "with", "precision"}, mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.Precision(), metrics.WithoutTimeout())

					sink.IncrCounter(ctx, []string{"counter", "without", "labels"}, float64(mock.GenObj[uint8]()),
						metrics.WithoutTimeout())

					sink.IncrCounter(ctx, []string{"counter", "with", "labels"},
						float64(mock.GenObj[uint8]()),
						metrics.Labels(labels),
						metrics.WithoutTimeout())

					sink.AddSample(ctx, []string{"sample", "without", "labels"}, mock.GenObj[float64](),
						metrics.WithoutTimeout())

					sink.AddSample(ctx, []string{"sample", "with", "labels"}, mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.WithoutTimeout())

					sink.AddSample(ctx, []string{"sample", "with", "buckets"}, mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)),
						metrics.WithoutTimeout())

					sink.AddSample(ctx, []string{"sample", "with", "precision"},
						mock.GenObj[float64](),
						metrics.Labels(labels),
						metrics.Precision(),
						metrics.WithoutTimeout())

					sink.MeasureSince(ctx, []string{"measure", "without", "labels"}, mock.GenObj[time.Time](),
						metrics.WithoutTimeout())

					sink.MeasureSince(ctx, []string{"measure", "with", "labels"},
						mock.GenObj[time.Time](),
						metrics.Labels(labels),
						metrics.WithoutTimeout())

					sink.MeasureSince(ctx, []string{"measure", "with", "buckets"},
						mock.GenObj[time.Time](),
						metrics.Labels(labels),
						metrics.PrometheusBuckets(prometheus.ExponentialBucketsRange(1, math.MaxInt64, 50)),
						metrics.WithoutTimeout())

					sink.MeasureSince(ctx, []string{"measure", "with", "precision"},
						mock.GenObj[time.Time](),
						metrics.Labels(labels),
						metrics.Precision(),
						metrics.WithoutTimeout())
				}
			}()
		}

		// Then
		wg.Wait()
		time.Sleep(timeout)
	})
}
