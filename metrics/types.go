package metrics

import (
	"context"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

const (
	ErrDuplicatedName utils.Error = "duplicated metrics name"
)

// The Sink interface is used to transmit metrics information
// to an external system
type Sink interface {
	// SetGauge A Gauge should retain the last value it is set to
	SetGauge(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender)

	// IncrCounter Counters should accumulate values
	IncrCounter(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender)

	// AddSample Samples are for timing information, where quantiles are used
	AddSample(ctx context.Context, key []string, val float64, opts ...utils.OptionExtender)

	// MeasureSince A better way to add timer samples
	MeasureSince(ctx context.Context, key []string, start time.Time, opts ...utils.OptionExtender)

	getProxy() any
	shutdown()
}

type Label struct {
	Key, Value string
}

// Conf metrics conf
//nolint: revive // struct tag too long issue
type Conf struct {
	Type                  metricsType       `yaml:"type" json:"type" toml:"type"`
	Mode                  mode              `yaml:"mode" json:"mode" toml:"mode"`
	Interval              string            `yaml:"interval" json:"interval" toml:"interval"`
	Endpoint              *endpointConf     `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	Labels                map[string]string `yaml:"labels" json:"labels" toml:"labels"`
	EnableServiceLabel    bool              `yaml:"enable_service_label" json:"enable_service_label" toml:"enable_service_label"`
	EnableRuntimeMetrics  bool              `yaml:"enable_runtime_metrics" json:"enable_runtime_metrics" toml:"enable_runtime_metrics"`
	EnableInternalMetrics bool              `yaml:"enable_internal_metrics" json:"enable_internal_metrics" toml:"enable_internal_metrics"`
	LogInstance           string            `yaml:"log_instance" json:"log_instance" toml:"log_instance"`
	QueueLimit            int               `yaml:"queue_limit" json:"queue_limit" toml:"queue_limit" default:"16384"`
	QueueConcurrency      int               `yaml:"queue_concurrency" json:"queue_concurrency" toml:"queue_concurrency"`
}

type endpointConf struct {
	Addresses []string `yaml:"addresses" json:"addresses" toml:"addresses"`
}

type metricsType string

const (
	metricsTypePrometheus metricsType = "prometheus"
)

type mode string

const (
	modePull mode = "pull"
	modePush mode = "push"
)

type cfg struct {
	c          *Conf
	ctx        context.Context
	name       string
	appName    string
	interval   time.Duration
	initOption *config.InitOption
}

type option struct {
	prometheusBuckets []float64
	precision         bool
	timeout           time.Duration
	labels            []Label
}

func PrometheusBuckets(buckets []float64) utils.OptionFunc[option] {
	return func(o *option) {
		o.prometheusBuckets = buckets
	}
}

func Precision() utils.OptionFunc[option] {
	return func(o *option) {
		o.precision = true
	}
}

func Timeout(timeout time.Duration) utils.OptionFunc[option] {
	return func(o *option) {
		o.timeout = timeout
	}
}

func WithoutTimeout() utils.OptionFunc[option] {
	return func(o *option) {
		o.timeout = -1
	}
}

func Labels(labels []Label) utils.OptionFunc[option] {
	return func(o *option) {

	}
}
