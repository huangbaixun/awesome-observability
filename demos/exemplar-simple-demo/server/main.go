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
	// init opentelemetry trace provider
	shutdown := initTracerProvider()
	defer shutdown()
	log.Println("init tracer provider done")

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		starttime := time.Now()

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
		traceId := trace.SpanContextFromContext(ctx).TraceID()
		log.Printf("found trace id %v\n", traceId.String())

		if _, err := w.Write([]byte("Hello World")); err != nil {
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
