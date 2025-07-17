package trace

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"

	semconv "go.opentelemetry.io/otel/semconv/v1.15.0"
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}
	for name, conf := range confs {
		addInstance(ctx, name, conf, opt)
	}

	return func() {
		rwlock.Lock()
		defer rwlock.Unlock()

		if appInstances != nil {
			delete(appInstances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	if utils.IsStrBlank(conf.ServiceName) {
		conf.ServiceName = opt.AppName
	}

	var exporter trace.SpanExporter
	switch conf.Type {
	case exporterTypeJaeger:
		exporter = utils.Must(newJaegerExporter(ctx, conf))
	case exporterTypeZipkin:
		exporter = utils.Must(newZipkinExporter(ctx, conf))
	case exporterTypeOTLP:
		exporter = utils.Must(newOTLPExporter(ctx, conf))
	case exporterTypeStdout:
		exporter = utils.Must(newStdoutExporter(ctx, conf))
	default:
		panic(ErrUnsupportedExporterType)
	}

	rs := utils.Must(resource.New(
		ctx,
		resource.WithOS(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithProcessPID(),
		resource.WithProcessOwner(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(conf.ServiceName),
			semconv.ServiceVersionKey.String(conf.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(conf.DeploymentEnv),
		),
	))

	tp := trace.NewTracerProvider(
		trace.WithSampler(buildSampler(conf)),
		trace.WithBatcher(exporter),
		trace.WithResource(rs),
	)

	rwlock.Lock()
	defer rwlock.Unlock()
	if appInstances == nil {
		appInstances = make(map[string]map[string]TracerProvider)
	}
	if appInstances[opt.AppName] == nil {
		appInstances[opt.AppName] = make(map[string]TracerProvider)
	}
	if _, ok := appInstances[opt.AppName][name]; ok {
		panic(ErrDuplicatedName)
	}
	appInstances[opt.AppName][name] = tp

	if name == config.DefaultInstanceKey {
		otel.SetTracerProvider(tp)
	}

	if opt.DI != nil {
		opt.DI.MustProvide(func() TracerProvider { return Use(name, AppName(opt.AppName)) }, di.Name(name))
	}
	if opt.App != nil {
		opt.App.MustProvide(
			func() TracerProvider { return Use(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}
}

func buildSampler(conf *Conf) trace.Sampler {
	switch conf.SampleType {
	case sampleTypeAlways:
		return trace.AlwaysSample()
	case sampleTypeNever:
		return trace.NeverSample()
	case sampleTypeTraceIDRatio:
		// 如果 ratio 无效，提供一个安全的默认值
		if !conf.SampleRatio.IsPositive() {
			return trace.NeverSample()
		}
		if conf.SampleRatio.GreaterThan(decimal.NewFromInt(1)) {
			return trace.AlwaysSample()
		}
		return trace.TraceIDRatioBased(conf.SampleRatio.InexactFloat64())
	default:
		panic(ErrUnsupportedSampleType)
	}
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func Use(name string, opts ...utils.OptionExtender) TracerProvider {
	opt := utils.ApplyOptions[useOption](opts...)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := appInstances[opt.appName]
	if !ok {
		panic(errors.Errorf("trace instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(errors.Errorf("kv trace not found for name: %s", name))
	}
	return instance
}

func init() {
	config.AddComponent(config.ComponentTrace, Construct, config.WithFlag(&flagString))
}
