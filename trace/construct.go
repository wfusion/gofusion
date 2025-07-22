package trace

import (
	"context"
	"log"
	"os"
	"reflect"
	"syscall"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func Construct(ctx context.Context, confs map[string]*Conf, opts ...utils.OptionExtender) func(context.Context) {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}
	for name, conf := range confs {
		addInstance(ctx, name, conf, opt)
	}

	return func(ctx context.Context) {
		rwlock.Lock()
		defer rwlock.Unlock()

		pid := syscall.Getpid()
		app := config.Use(opt.AppName).AppName()
		if appInstances != nil {
			for name, instance := range appInstances[opt.AppName] {
				if err := instance.shutdown(ctx); err != nil {
					log.Printf("%v [Gofusion] %s %s %s shutdown error: %s",
						pid, app, config.ComponentTrace, name, err)
				}
			}
			delete(appInstances, opt.AppName)
		}
	}
}

func addInstance(ctx context.Context, name string, conf *Conf, opt *config.InitOption) {
	if utils.IsStrBlank(conf.ServiceName) {
		conf.ServiceName = opt.AppName
	}

	exporter := buildExporter(ctx, conf)
	opts := []trace.TracerProviderOption{trace.WithResource(buildResource(ctx, conf))}
	if !conf.EnableBatchExporter {
		opts = append(opts, trace.WithSyncer(exporter))
	} else {
		opts = append(opts, trace.WithBatcher(exporter,
			trace.WithBatchTimeout(conf.BatchExporterConf.BatchTimeout.Duration),
			trace.WithMaxExportBatchSize(conf.BatchExporterConf.MaxExportBatchSize),
			trace.WithMaxQueueSize(conf.BatchExporterConf.MaxQueueSize),
			trace.WithExportTimeout(conf.BatchExporterConf.ExportTimeout.Duration),
		))
	}

	var sampler trace.Sampler
	if utils.IsStrBlank(conf.Sampler) {
		sampler = buildSampler(&conf.Sample)
	} else {
		val := reflect.New(inspect.TypeOf(conf.Sampler))
		if val.Type().Implements(customSamplerType) {
			utils.MustSuccess(val.Interface().(customSampler).Init(ctx, conf))
		}
		sampler = val.Interface().(trace.Sampler)
	}
	opts = append(opts, trace.WithSampler(sampler))

	if utils.IsStrNotBlank(conf.IDGenerator) {
		val := reflect.New(inspect.TypeOf(conf.IDGenerator))
		if val.Type().Implements(customIDGeneratorType) {
			utils.MustSuccess(val.Interface().(customIDGenerator).Init(ctx, conf))
		}
		opts = append(opts, trace.WithIDGenerator(val.Interface().(trace.IDGenerator)))
	}

	tp := newTraceProvider(ctx, name, conf, trace.NewTracerProvider(opts...), exporter)
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
		if utils.IsStrNotBlank(conf.TextMapPropagator) {
			val := reflect.New(inspect.TypeOf(conf.TextMapPropagator))
			if val.Type().Implements(customPropagatorType) {
				utils.MustSuccess(val.Interface().(customPropagator).Init(ctx, conf))
			}
			otel.SetTextMapPropagator(val.Interface().(propagation.TextMapPropagator))
		}
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

// buildResource custom overrides env overrides config overrides legacy
func buildResource(ctx context.Context, conf *Conf) *resource.Resource {
	legacyResource := utils.Must(
		resource.New(
			ctx,
			resource.WithSchemaURL(semconv.SchemaURL),
			resource.WithOS(),
			resource.WithHost(),
			resource.WithContainer(),
			resource.WithProcessPID(),
			resource.WithProcessOwner(),
			resource.WithTelemetrySDK(),
		),
	)
	configResource := resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
		semconv.HostNameKey.String(utils.Must(os.Hostname())),
		semconv.ServiceNameKey.String(conf.ServiceName),
		semconv.ServiceVersionKey.String(conf.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(conf.DeploymentEnv),
		semconv.ProcessCommandKey.String(os.Args[0]),
		attribute.Key("host.ip").String(utils.NonDefaultLocalIP()),
	}...)
	mergedResource := utils.Must(resource.Merge(legacyResource, configResource))

	attrs := make([]attribute.KeyValue, 0, len(conf.CustomResources))
	for _, v := range resource.Environment().Attributes() {
		attrs = append(attrs, v)
	}
	envResource := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	mergedResource = utils.Must(resource.Merge(mergedResource, envResource))

	attrs = attrs[:0]
	for k, v := range conf.CustomResources {
		attrs = append(attrs, attribute.Key(k).String(v))
	}
	customResource := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	mergedResource = utils.Must(resource.Merge(mergedResource, customResource))

	return mergedResource
}

func buildExporter(ctx context.Context, conf *Conf) (exporter trace.SpanExporter) {
	switch conf.Type {
	case exporterTypeJaeger:
		return utils.Must(newJaegerExporter(ctx, conf))
	case exporterTypeZipkin:
		return utils.Must(newZipkinExporter(ctx, conf))
	case exporterTypeOTLP:
		return utils.Must(newOTLPExporter(ctx, conf))
	case exporterTypeStdout:
		return utils.Must(newStdoutExporter(ctx, conf))
	case exporterTypeCustom:
		val := reflect.New(inspect.TypeOf(conf.Exporter))
		if val.Type().Implements(customSpanExporterType) {
			utils.MustSuccess(val.Interface().(customSpanExporter).Init(ctx, conf))
		}
		return val.Interface().(trace.SpanExporter)
	default:
		panic(ErrUnsupportedExporterType)
	}
}

func buildSampler(conf *SampleConf) (sampler trace.Sampler) {
	if conf == nil {
		return
	}

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
	case sampleTypeParentBased:
		rootSampler := buildSampler(conf.ParentBased.RootSample)
		return trace.ParentBased(rootSampler, []trace.ParentBasedSamplerOption{
			trace.WithRemoteParentSampled(buildSampler(conf.ParentBased.RemoteParentSampled)),
			trace.WithRemoteParentNotSampled(buildSampler(conf.ParentBased.RemoteParentNotSampled)),
			trace.WithLocalParentSampled(buildSampler(conf.ParentBased.LocalParentSampled)),
			trace.WithLocalParentNotSampled(buildSampler(conf.ParentBased.LocalParentNotSampled)),
		}...)
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

func NewDI(name string, opts ...utils.OptionExtender) func() TracerProvider {
	return func() TracerProvider {
		return Use(name, opts...)
	}
}

func Internal(opts ...utils.OptionExtender) (traces []TracerProvider) {
	opt := utils.ApplyOptions[useOption](opts...)
	appName := config.Use(opt.appName).AppName()
	rwlock.Lock()
	defer rwlock.Unlock()
	tps, ok := appInstances[appName]
	if !ok {
		return
	}
	for _, tp := range tps {
		if tp.config().EnableInternalTrace {
			traces = append(traces, tp)
		}
	}
	return
}

func init() {
	config.AddComponent(config.ComponentTrace, Construct, config.WithFlag(&flagString))
}
