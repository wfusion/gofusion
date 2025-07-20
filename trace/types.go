package trace

import (
	"reflect"

	"github.com/shopspring/decimal"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	ErrDuplicatedName          utils.Error = "duplicated trace name"
	ErrUnsupportedExporterType utils.Error = "unsupported trace exporter type"
	ErrUnsupportedProtocolType utils.Error = "unsupported trace OTLP protocol type"
	ErrUnsupportedSampleType   utils.Error = "unsupported trace sample type"
)

var (
	customSamplerType      = reflect.TypeOf((*customSampler)(nil)).Elem()
	customPropagatorType   = reflect.TypeOf((*customPropagator)(nil)).Elem()
	customIDGeneratorType  = reflect.TypeOf((*customIDGenerator)(nil)).Elem()
	customSpanExporterType = reflect.TypeOf((*customSpanExporter)(nil)).Elem()
)

type Conf struct {
	Type                exporterType      `yaml:"type" json:"type" toml:"type"`
	ServiceName         string            `yaml:"service_name" json:"service_name" toml:"service_name"`
	ServiceVersion      string            `yaml:"service_version" json:"service_version" toml:"service_version"`
	DeploymentEnv       string            `yaml:"deployment_env" json:"deployment_env" toml:"deployment_env"`
	EndpointConf        EndpointConf      `yaml:"endpoint_conf" json:"endpoint_conf" toml:"endpoint_conf"`
	Sample              SampleConf        `yaml:"sample" json:"sample" toml:"sample"`
	PrettyPrint         bool              `yaml:"pretty_print" json:"pretty_print" toml:"pretty_print"`
	EnableBatchExporter bool              `yaml:"enable_batch_exporter" json:"enable_batch_exporter" toml:"enable_batch_exporter"`
	BatchExporterConf   BatchExporterConf `yaml:"batch_exporter_conf" json:"batch_exporter_conf" toml:"batch_exporter_conf"`
	Sampler             string            `yaml:"sampler" json:"sampler" toml:"sampler"`
	Exporter            string            `yaml:"exporter" json:"exporter" toml:"exporter"`
	TextMapPropagator   string            `yaml:"text_map_propagator" json:"text_map_propagator" toml:"text_map_propagator"`
	IDGenerator         string            `yaml:"id_generator" json:"id_generator" toml:"id_generator"`
	CustomResources     map[string]string `yaml:"custom_resources" json:"custom_resources" toml:"custom_resources"`
	EnableInternalTrace bool              `yaml:"enable_internal_trace" json:"enable_internal_trace" toml:"enable_internal_trace"`
	CustomProps         map[string]any    `yaml:"custom_props" json:"custom_props" toml:"custom_props"`
}

type EndpointConf struct {
	Username           string            `yaml:"username" json:"username" toml:"username"`
	Password           string            `yaml:"password" json:"password" toml:"password" encrypted:""`
	Endpoint           string            `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	OTLPProtocol       otlpProtocolType  `yaml:"otlp_protocol" json:"otlp_protocol" toml:"otlp_protocol"`
	OTLPInsecure       bool              `yaml:"otlp_insecure" json:"otlp_insecure" toml:"otlp_insecure"`
	OTLPEnableCompress bool              `yaml:"otlp_enable_compress" json:"otlp_enable_compress" toml:"otlp_enable_compress"`
	OTLPTLSCAFile      string            `yaml:"otlp_tls_ca_file" json:"otlp_tls_ca_file" toml:"otlp_tls_ca_file"`
	OTLPTLSCertFile    string            `yaml:"otlp_tls_cert_file" json:"otlp_tls_cert_file" toml:"otlp_tls_cert_file"`
	OTLPTLSKeyFile     string            `yaml:"otlp_tls_key_file" json:"otlp_tls_key_file" toml:"otlp_tls_key_file"`
	OTLPTLSCACert      string            `yaml:"otlp_tls_ca_cert" json:"otlp_tls_ca_cert" toml:"otlp_tls_ca_cert"`
	OTLPTLSClientCert  string            `yaml:"otlp_tls_client_cert" json:"otlp_tls_client_cert" toml:"otlp_tls_client_cert"`
	OTLPTLSClientKey   string            `yaml:"otlp_tls_client_key" json:"otlp_tls_client_key" toml:"otlp_tls_client_key"`
	OTLPTLSServerName  string            `yaml:"otlp_tls_server_name" json:"otlp_tls_server_name" toml:"otlp_tls_server_name"`
	OTLPHeaders        map[string]string `yaml:"otlp_headers" json:"otlp_headers" toml:"otlp_headers"`
	OTLPTimeout        utils.Duration    `yaml:"otlp_timeout" json:"otlp_timeout" toml:"otlp_timeout" default:"10s"`
}

type SampleConf struct {
	SampleType  sampleType         `yaml:"type" json:"type" toml:"type"`
	SampleRatio decimal.Decimal    `yaml:"ratio" json:"ratio" toml:"ratio"`
	ParentBased *ParentBasedSample `yaml:"parent_based" json:"parent_based" toml:"parent_based"`
}

type ParentBasedSample struct {
	RootSample             *SampleConf `yaml:"root_sample" json:"root_sample" toml:"root_sample"`
	RemoteParentSampled    *SampleConf `yaml:"remote_parent_sampled" json:"remote_parent_sampled" toml:"remote_parent_sampled"`
	RemoteParentNotSampled *SampleConf `yaml:"remote_parent_not_sampled" json:"remote_parent_not_sampled" toml:"remote_parent_not_sampled"`
	LocalParentSampled     *SampleConf `yaml:"local_parent_sampled" json:"local_parent_sampled" toml:"local_parent_sampled"`
	LocalParentNotSampled  *SampleConf `yaml:"local_parent_not_sampled" json:"local_parent_not_sampled" toml:"local_parent_not_sampled"`
}

type BatchExporterConf struct {
	// MaxQueueSize is the maximum queue size to buffer spans for delayed processing. If the
	// queue gets full it drops the spans. Use BlockOnQueueFull to change this behavior.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int `yaml:"max_queue_size" json:"max_queue_size" toml:"max_queue_size" default:"2048"`

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available spans when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout utils.Duration `yaml:"batch_timeout" json:"batch_timeout" toml:"batch_timeout" default:"5s"`

	// ExportTimeout specifies the maximum duration for exporting spans. If the timeout
	// is reached, the export will be cancelled.
	// The default value of ExportTimeout is 30000 msec.
	ExportTimeout utils.Duration `yaml:"export_timeout" json:"export_timeout" toml:"export_timeout" default:"30s"`

	// MaxExportBatchSize is the maximum number of spans to process in a single batch.
	// If there are more than one batch worth of spans then it processes multiple batches
	// of spans one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int `yaml:"max_export_batch_size" json:"max_export_batch_size" toml:"max_export_batch_size" default:"512"`

	// BlockOnQueueFull blocks onEnd() and onStart() method if the queue is full
	// AND if BlockOnQueueFull is set to true.
	// Blocking option should be used carefully as it can severely affect the performance of an
	// application.
	BlockOnQueueFull bool `yaml:"block_on_queue_full" json:"block_on_queue_full" toml:"block_on_queue_full"`
}

type exporterType string

const (
	exporterTypeJaeger exporterType = "jaeger"
	exporterTypeZipkin exporterType = "zipkin"
	exporterTypeOTLP   exporterType = "otlp"
	exporterTypeStdout exporterType = "stdout"
	exporterTypeCustom exporterType = "custom"
)

type otlpProtocolType string

const (
	protocolTypeHTTP otlpProtocolType = "http"
	protocolTypeGRPC otlpProtocolType = "grpc"
)

type sampleType string

const (
	sampleTypeAlways       sampleType = "always"
	sampleTypeNever        sampleType = "never"
	sampleTypeTraceIDRatio sampleType = "trace_id_ratio"
	sampleTypeParentBased  sampleType = "parent_based"
)
