package main

import (
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

func main() {
	time.Sleep(3 * time.Second)
	// init opentelemetry trace provider
	shutdown := initTracerProvider()
	defer shutdown()
	log.Println("init tracer provider done")

	var count int = 0
	var sleep float64

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		starttime := time.Now()
		count++

		if count%100 == 0 {
			sleep = 1
		} else {
			sleep = rng.Float64() / 10
		}
		time.Sleep(time.Duration(sleep) * time.Second)

		ctx := req.Context()
		traceId := trace.SpanContextFromContext(ctx).TraceID()

		if _, err := w.Write([]byte("world")); err != nil {
			http.Error(w, "write operation failed.", http.StatusInternalServerError)
			return
		}

		log.Printf("request use %v seconds\n", float64(time.Since(starttime))/float64(time.Second))
		requestDurationsHistogram.(prometheus.ExemplarObserver).ObserveWithExemplar(float64(time.Since(starttime))/float64(time.Second), prometheus.Labels{
			"traceID": traceId.String(),
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
		log.Fatalf("server failed to serve: %v\n", err)
	}
}
