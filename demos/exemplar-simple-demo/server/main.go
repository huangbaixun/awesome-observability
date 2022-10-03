package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

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

	log.Println("start server at http://0.0.0.0:7777")
	server := &http.Server{
		Addr:    "0.0.0.0:7777",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		handleErr(err, "server failed to serve")
	}
}
