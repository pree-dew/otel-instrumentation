package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ShutdownFunc is a delegate that shuts down the OpenTelemetry components.
type ShutdownFunc func(ctx context.Context) error

// Run sets the global OpenTelemetry tracer provider and meter provider
// configured to use the OTLP HTTP exporter that will send telemetry
// to a local OpenTelemetry Collector.
func Run(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	// Initialized the returned shutdownFunc to no-op.
	shutdownFunc := func(ctx context.Context) error { return nil }

	// Create Resource.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return shutdownFunc, err
	}

	// Create the OTLP exporters.
	traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		return shutdownFunc, err
	}

	// Create the TracerProvider.
	// A provider is an implementation of the OpenTelemetry instrumentation API. These providers handle all of the API calls
	// TraceProvider creates tracers and spans
	tp := trace.NewTracerProvider(
		// Record information about this application in an Resource.
		trace.WithResource(res),
		// Set traces exporter.
		trace.WithBatcher(traceExp),
	)

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	// Register W3C Trace Context propagator as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Update the returned shutdownFunc that calls both providers'
	// shutdown methods and make sure that a non-nil error is returned
	// if any returneed an error.
	shutdownFunc = func(ctx context.Context) error {
		var retErr error
		if err := tp.Shutdown(ctx); err != nil {
			retErr = err
		}
		return retErr
	}

	// Return the Shutdown function so that it can be used by the caller to
	// send all the telemetry before the application closes.
	return shutdownFunc, nil
}
