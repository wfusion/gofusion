package trace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"

	"github.com/wfusion/gofusion/common/utils"
)

func newZipkinExporter(ctx context.Context, conf *Conf) (exporter trace.SpanExporter, err error) {
	return zipkin.New(conf.EndpointConf.Endpoint)
}

func newJaegerExporter(ctx context.Context, conf *Conf) (exporter trace.SpanExporter, err error) {
	opts := []jaeger.CollectorEndpointOption{
		jaeger.WithEndpoint(conf.EndpointConf.Endpoint),
	}

	if utils.IsStrNotBlank(conf.EndpointConf.Username) {
		opts = append(opts, jaeger.WithUsername(conf.EndpointConf.Username))
	}
	if utils.IsStrNotBlank(conf.EndpointConf.Password) {
		opts = append(opts, jaeger.WithPassword(conf.EndpointConf.Password))
	}

	return jaeger.New(jaeger.WithCollectorEndpoint(opts...))
}

func newOTLPExporter(ctx context.Context, conf *Conf) (exporter trace.SpanExporter, err error) {
	switch conf.EndpointConf.Protocol {
	case protocolTypeHTTP:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(conf.EndpointConf.Endpoint),
		}
		if conf.EndpointConf.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	case protocolTypeGRPC:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(conf.EndpointConf.Endpoint),
		}
		if conf.EndpointConf.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else {
			var tlsCfg *tls.Config
			if tlsCfg, err = buildOTLPGrpcTLSConfig(&conf.EndpointConf); err != nil {
				return
			}
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, ErrUnsupportedProtocolType
	}
}

func newStdoutExporter(ctx context.Context, conf *Conf) (exporter trace.SpanExporter, err error) {
	opts := []stdouttrace.Option{
		stdouttrace.WithWriter(os.Stdout),
	}
	if conf.PrettyPrint {
		opts = append(opts, stdouttrace.WithPrettyPrint())
	}
	return stdouttrace.New(opts...)
}

func buildOTLPGrpcTLSConfig(conf *EndpointConf) (cfg *tls.Config, err error) {
	var cp *x509.CertPool
	if utils.IsStrNotBlank(conf.TLSCACert) || utils.IsStrNotBlank(conf.TLSClientCert) {
		cp, _ = x509.SystemCertPool()
		if cp == nil {
			cp = x509.NewCertPool()
		}

		caBytes := []byte(conf.TLSCACert)
		if len(caBytes) == 0 {
			if caBytes, err = os.ReadFile(conf.TLSCAFile); err != nil {
				return
			}
		}

		if !cp.AppendCertsFromPEM(caBytes) {
			return nil, errors.New("failed to append CA certificate")
		}
	}

	var certList []tls.Certificate
	if utils.IsStrNotBlank(conf.TLSClientCert) && utils.IsStrNotBlank(conf.TLSClientKey) {
		cert, err := tls.X509KeyPair([]byte(conf.TLSClientCert), []byte(conf.TLSClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load client key pair: %w", err)
		}
		certList = append(certList, cert)
	}

	if utils.IsStrNotBlank(conf.TLSCertFile) && utils.IsStrNotBlank(conf.TLSKeyFile) {
		cert, err := tls.LoadX509KeyPair(conf.TLSCertFile, conf.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client key pair: %w", err)
		}
		certList = append(certList, cert)
	}

	cfg = &tls.Config{
		RootCAs:      cp,
		Certificates: certList,
		ServerName:   conf.TLSServerName,
	}
	return
}
