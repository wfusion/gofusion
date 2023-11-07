package metrics

import (
	"context"
	"reflect"
	"time"

	"github.com/wfusion/gofusion/common/infra/metrics"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedName utils.Error = "duplicated metrics name"
)

var (
	customLoggerType  = reflect.TypeOf((*customLogger)(nil)).Elem()
	metricsLoggerType = reflect.TypeOf((*metrics.Logger)(nil)).Elem()
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

	// IsEnableServiceLabel check if enable service label
	IsEnableServiceLabel() bool

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
	EnableInternalMetrics bool              `yaml:"enable_internal_metrics" json:"enable_internal_metrics" toml:"enable_internal_metrics"`
	QueueLimit            int               `yaml:"queue_limit" json:"queue_limit" toml:"queue_limit" default:"16384"`
	QueueConcurrency      int               `yaml:"queue_concurrency" json:"queue_concurrency" toml:"queue_concurrency"`
	Logger                string            `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.metricsLogger"`
	LogInstance           string            `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
	EnableLogger          bool              `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
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
	logger     metrics.Logger
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
		o.labels = labels
	}
}

type customLogger interface {
	Init(log log.Logable, appName, name string)
}
