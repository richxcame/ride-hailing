package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the configuration for the tracer
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	SampleRate     float64
	Enabled        bool
}

// TracerProvider holds the global tracer provider
var globalTracerProvider *sdktrace.TracerProvider

// InitTracer initializes the OpenTelemetry tracer with OTLP exporter
func InitTracer(cfg Config, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		logger.Info("Tracing is disabled")
		return nil, nil
	}

	ctx := context.Background()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
			attribute.String("host.name", getHostname()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up OTLP trace exporter
	otlpEndpoint := cfg.OTLPEndpoint
	if otlpEndpoint == "" {
		otlpEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	}

	logger.Info("Initializing OTLP trace exporter",
		zap.String("endpoint", otlpEndpoint),
		zap.String("service", cfg.ServiceName),
		zap.Float64("sample_rate", cfg.SampleRate),
	)

	// Create gRPC connection to the collector
	conn, err := grpc.NewClient(
		otlpEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// Create OTLP trace exporter
	traceExporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithGRPCConn(conn),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Configure sampling strategy
	sampler := configureSampler(cfg)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator to W3C Trace Context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	globalTracerProvider = tp

	logger.Info("OpenTelemetry tracer initialized successfully",
		zap.String("service", cfg.ServiceName),
		zap.String("endpoint", otlpEndpoint),
	)

	return tp, nil
}

// configureSampler configures the sampling strategy based on environment and config
func configureSampler(cfg Config) sdktrace.Sampler {
	sampleRate := cfg.SampleRate
	if sampleRate <= 0 {
		// Check environment variable for override
		if envRate := getEnv("OTEL_TRACE_SAMPLE_RATE", ""); envRate != "" {
			fmt.Sscanf(envRate, "%f", &sampleRate)
		}
	}

	// Default sample rates based on environment
	if sampleRate <= 0 {
		switch cfg.Environment {
		case "development", "dev", "local":
			sampleRate = 1.0 // 100% in development
		case "staging", "stage":
			sampleRate = 0.5 // 50% in staging
		case "production", "prod":
			sampleRate = 0.1 // 10% in production
		default:
			sampleRate = 1.0
		}
	}

	// Always sample error spans
	return sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(sampleRate),
		sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),
		sdktrace.WithRemoteParentNotSampled(sdktrace.TraceIDRatioBased(sampleRate)),
	)
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	if globalTracerProvider == nil {
		return nil
	}
	return globalTracerProvider.Shutdown(ctx)
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetTracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError records an error in the current span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, trace.WithAttributes(attrs...))
	}
}

// GetTraceID returns the trace ID from the context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
