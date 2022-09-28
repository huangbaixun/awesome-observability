package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func initOtlpSpanExporter() (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	// parse env config
	otelCollectorAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelCollectorAddr = "0.0.0.0:4317"
	}

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
	spanExporter, err := initStdoutSpanExporter()
	if err != nil {
		log.Fatalf("init span exporter failed: %v\n", err)
	}

	// res, err := resource.New(ctx,
	// 	resource.WithFromEnv(),
	// 	resource.WithProcess(),
	// 	resource.WithTelemetrySDK(),
	// 	resource.WithHost(),
	// 	resource.WithAttributes(
	// 		// the service name used to display traces in backends
	// 		semconv.ServiceNameKey.String("demo-server"),
	// 	),
	// )
	// handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(spanExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		// sdktrace.WithResource(res),
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

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

var (
	requestDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_durations_histogram_seconds",
		Help:    "HTTP request latency distributions.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	})
)

func main() {
	shutdown := initTracerProvider()
	defer shutdown()
	log.Println("init tracer provider done")

	// meter := global.Meter("demo-server-meter")
	// serverAttribute := attribute.String("server-attribute", "foo")
	// commonLabels := []attribute.KeyValue{serverAttribute}
	// requestCount, _ := meter.SyncInt64().Counter(
	// 	"demo_server/request_counts",
	// 	instrument.WithDescription("The number of requests received"),
	// )

	log.Println("registry handler")
	// create a handler wrapped in OpenTelemetry instrumentation
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		//  random sleep to simulate latency
		starttime := time.Now()

		fmt.Println("hello world")

		var sleep int64

		switch modulus := time.Now().Unix() % 5; modulus {
		case 0:
			sleep = rng.Int63n(2000)
		case 1:
			sleep = rng.Int63n(15)
		case 2:
			sleep = rng.Int63n(917)
		case 3:
			sleep = rng.Int63n(87)
		case 4:
			sleep = rng.Int63n(1173)
		}
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		ctx := req.Context()
		// requestCount.Add(ctx, 1, commonLabels...)
		// span := trace.SpanFromContext(ctx)
		traceId := trace.SpanContextFromContext(ctx).TraceID()
		log.Printf("found trace id %v\n", traceId.String())
		// bag := baggage.FromContext(ctx)

		// var baggageAttributes []attribute.KeyValue
		// baggageAttributes = append(baggageAttributes, serverAttribute)
		// for _, member := range bag.Members() {
		// 	baggageAttributes = append(baggageAttributes, attribute.String("baggage key:"+member.Key(), member.Value()))
		// }
		// span.SetAttributes(baggageAttributes...)

		if _, err := w.Write([]byte("Hello World")); err != nil {
			http.Error(w, "write operation failed.", http.StatusInternalServerError)
			return
		}

		requestDurationsHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(float64(time.Since(starttime)), prometheus.Labels{
			"TraceID": traceId.String(),
		})
	})

	mux := http.NewServeMux()
	mux.Handle("/hello", otelhttp.NewHandler(handler, "/hello"))

	prometheus.Register(requestDurationsHistogram)
	mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	log.Println("start server")
	server := &http.Server{
		Addr:    "127.0.0.1:7777",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		handleErr(err, "server failed to serve")
	}
}
