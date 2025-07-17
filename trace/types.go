package trace

import (
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/trace"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	ErrDuplicatedName          utils.Error = "duplicated trace name"
	ErrUnsupportedExporterType utils.Error = "unsupported trace exporter type"
	ErrUnsupportedProtocolType utils.Error = "unsupported trace OTLP protocol type"
	ErrUnsupportedSampleType   utils.Error = "unsupported trace sample type"
)

type TracerProvider interface {
	trace.TracerProvider
}

type Conf struct {
	Type           exporterType    `yaml:"type" json:"type" toml:"type"`
	ServiceName    string          `yaml:"service_name" json:"service_name" toml:"service_name"`
	ServiceVersion string          `yaml:"service_version" json:"service_version" toml:"service_version"`
	DeploymentEnv  string          `yaml:"deployment_env" json:"deployment_env" toml:"deployment_env"`
	EndpointConf   EndpointConf    `yaml:"endpoint_conf" json:"endpoint_conf" toml:"endpoint_conf"`
	SampleType     sampleType      `yaml:"sample_type" json:"sample_type" toml:"sample_type"`
	SampleRatio    decimal.Decimal `yaml:"sample_ratio" json:"sample_ratio" toml:"sample_ratio"`
	PrettyPrint    bool            `yaml:"pretty_print" json:"pretty_print" toml:"pretty_print"`
}

type EndpointConf struct {
	Username      string       `yaml:"username" json:"username" toml:"username"`
	Password      string       `yaml:"password" json:"password" toml:"password" encrypted:""`
	Endpoint      string       `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	Protocol      protocolType `yaml:"protocol" json:"protocol" toml:"protocol"`
	Insecure      bool         `yaml:"insecure" json:"insecure" toml:"insecure"`
	TLSCAFile     string       `yaml:"tls_ca_file" json:"tls_ca_file" toml:"tls_ca_file"`
	TLSCertFile   string       `yaml:"tls_cert_file" json:"tls_cert_file" toml:"tls_cert_file"`
	TLSKeyFile    string       `yaml:"tls_key_file" json:"tls_key_file" toml:"tls_key_file"`
	TLSCACert     string       `yaml:"tls_ca_cert" json:"tls_ca_cert" toml:"tls_ca_cert"`
	TLSClientCert string       `yaml:"tls_client_cert" json:"tls_client_cert" toml:"tls_client_cert"`
	TLSClientKey  string       `yaml:"tls_client_key" json:"tls_client_key" toml:"tls_client_key"`
	TLSServerName string       `yaml:"tls_server_name" json:"tls_server_name" toml:"tls_server_name"`
}

type exporterType string

const (
	exporterTypeJaeger exporterType = "jaeger"
	exporterTypeZipkin exporterType = "zipkin"
	exporterTypeOTLP   exporterType = "otlp"
	exporterTypeStdout exporterType = "stdout"
)

type protocolType string

const (
	protocolTypeHTTP protocolType = "http"
	protocolTypeGRPC protocolType = "grpc"
)

type sampleType string

const (
	sampleTypeAlways       sampleType = "always"
	sampleTypeNever        sampleType = "never"
	sampleTypeTraceIDRatio sampleType = "trace_id_ratio"
)
