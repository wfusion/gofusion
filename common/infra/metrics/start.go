package metrics

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfusion/gofusion/common/utils"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// Config is used to configure metrics settings
type Config struct {
	ServiceName         string        // Prefixed with keys to separate services
	HostName            string        // Hostname to use. If not provided and EnableHostname, it will be os.Hostname
	EnableHostname      bool          // Enable prefixing gauge values with hostname
	EnableHostnameLabel bool          // Enable adding hostname to labels
	EnableServiceLabel  bool          // Enable adding service to labels
	EnableClientIPLabel bool          // Enable adding service ip to labels
	EnableTypePrefix    bool          // Prefixes key with a type ("counter", "gauge", "timer")
	TimerGranularity    time.Duration // Granularity of timers.
	ProfileInterval     time.Duration // Interval to profile runtime metrics

	AllowedPrefixes []string // A list of metric prefixes to allow, with '.' as the separator
	BlockedPrefixes []string // A list of metric prefixes to block, with '.' as the separator
	AllowedLabels   []string // A list of metric labels to allow, with '.' as the separator
	BlockedLabels   []string // A list of metric labels to block, with '.' as the separator
	FilterDefault   bool     // Whether to allow metrics by default
}

// Metrics represents an instance of a metrics sink that can
// be used to emit
type Metrics struct {
	Config
	sink          MetricSink
	filter        *iradix.Tree
	allowedLabels map[string]bool
	blockedLabels map[string]bool
	filterLock    sync.RWMutex // Lock filters and allowedLabels/blockedLabels access
}

// Shared global metrics instance
var globalMetrics atomic.Value // *Metrics

func init() {
	// Initialize to a blackhole sink to avoid errors
	globalMetrics.Store(&Metrics{sink: &BlackholeSink{}})
}

// Default returns the shared global metrics instance.
func Default() *Metrics {
	return globalMetrics.Load().(*Metrics)
}

// DefaultConfig provides a sane default configuration
func DefaultConfig(serviceName string) *Config {
	c := &Config{
		ServiceName:      serviceName, // Use client provided service
		HostName:         "",
		EnableHostname:   true,             // Enable hostname prefix
		EnableTypePrefix: false,            // Disable type prefix
		TimerGranularity: time.Millisecond, // Timers are in milliseconds
		ProfileInterval:  time.Second,      // Poll runtime every second
		FilterDefault:    true,             // Don't filter metrics by default
	}

	// Try to get the hostname
	name, _ := os.Hostname()
	c.HostName = name
	return c
}

// New is used to create a new instance of Metrics
func New(conf *Config, sink MetricSink, opts ...utils.OptionExtender) (*Metrics, error) {
	met := &Metrics{}
	met.Config = *conf
	met.sink = sink
	met.UpdateFilterAndLabels(conf.AllowedPrefixes, conf.BlockedPrefixes, conf.AllowedLabels, conf.BlockedLabels)

	return met, nil
}

// NewGlobal is the same as New, but it assigns the metrics object to be
// used globally as well as returning it.
func NewGlobal(conf *Config, sink MetricSink) (*Metrics, error) {
	metrics, err := New(conf, sink)
	if err == nil {
		globalMetrics.Store(metrics)
	}
	return metrics, err
}

// Proxy all the methods to the globalMetrics instance

// SetGauge Set gauge key and value with 32 bit precision
func SetGauge(key []string, val float32, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).SetGauge(key, val, opts...)
}

// SetGaugeWithLabels Set gauge key and value with 32 bit precision
func SetGaugeWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).SetGaugeWithLabels(key, val, labels, opts...)
}

// SetPrecisionGauge Set gauge key and value with 64 bit precision
// The Sink needs to implement PrecisionGaugeMetricSink, in case it doesn't,
// the metric value won't be set and ingored instead
func SetPrecisionGauge(key []string, val float64, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).SetPrecisionGauge(key, val, opts...)
}

// SetPrecisionGaugeWithLabels Set gauge key, value with 64 bit precision, and labels
// The Sink needs to implement PrecisionGaugeMetricSink, in case it doesn't,
// the metric value won't be set and ingored instead
func SetPrecisionGaugeWithLabels(key []string, val float64, labels []Label, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).SetPrecisionGaugeWithLabels(key, val, labels, opts...)
}

func EmitKey(key []string, val float32, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).EmitKey(key, val, opts...)
}

func IncrCounter(key []string, val float32, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).IncrCounter(key, val, opts...)
}

func IncrCounterWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).IncrCounterWithLabels(key, val, labels, opts...)
}

func AddSample(key []string, val float32, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).AddSample(key, val, opts...)
}

func AddSampleWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).AddSampleWithLabels(key, val, labels, opts...)
}

func MeasureSince(key []string, start time.Time, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).MeasureSince(key, start, opts...)
}

func MeasureSinceWithLabels(key []string, start time.Time, labels []Label, opts ...utils.OptionExtender) {
	globalMetrics.Load().(*Metrics).MeasureSinceWithLabels(key, start, labels, opts...)
}

func UpdateFilter(allow, block []string) {
	globalMetrics.Load().(*Metrics).UpdateFilter(allow, block)
}

// UpdateFilterAndLabels set allow/block prefixes of metrics while allowedLabels
// and blockedLabels - when not nil - allow filtering of labels in order to
// block/allow globally labels (especially useful when having large number of
// values for a given label). See README.md for more information about usage.
func UpdateFilterAndLabels(allow, block, allowedLabels, blockedLabels []string) {
	globalMetrics.Load().(*Metrics).UpdateFilterAndLabels(allow, block, allowedLabels, blockedLabels)
}

// Shutdown disables metric collection, then blocks while attempting to flush metrics to storage.
// WARNING: Not all MetricSink backends support this functionality, and calling this will cause them to leak resources.
// This is intended for use immediately prior to application exit.
func Shutdown() {
	m := globalMetrics.Load().(*Metrics)
	// Swap whatever MetricSink is currently active with a BlackholeSink. Callers must not have a
	// reason to expect that calls to the library will successfully collect metrics after Shutdown
	// has been called.
	globalMetrics.Store(&Metrics{sink: &BlackholeSink{}})
	m.Shutdown()
}
