package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc"
)

func initOtlpSpanExporter() (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	// parse env config
	otelCollectorAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelCollectorAddr = "0.0.0.0:4317"
	}

	log.Printf("otel collector addr: %v\n", otelCollectorAddr)

	// initialize grpc client
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelCollectorAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))

	// create span exporter
	sctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return otlptrace.New(sctx, traceClient)
}

func initStdoutSpanExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

func initTracerProvider() func() {
	ctx := context.Background()
	spanExporter, err := initOtlpSpanExporter()
	if err != nil {
		log.Fatalf("init span exporter failed: %v\n", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("demo-client"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v\n", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(spanExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	// otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}
