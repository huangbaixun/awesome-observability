package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestDurationHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "http request latency distributions.",
	})
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	prometheus.MustRegister(httpRequestDurationHistogram)
}

func main() {
	var count int = 0
	var sleep float64

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		count++
		if count%10 == 0 {
			sleep = 5
		} else {
			sleep = rng.Float64() / 10
		}

		time.Sleep(time.Duration(sleep) * time.Second)

		if _, err := w.Write([]byte("pong")); err != nil {
			http.Error(w, "write operation failed.", http.StatusInternalServerError)
			return
		}
		httpRequestDurationHistogram.Observe(sleep)
	})

	http.Handle("/ping", handler)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(":8888", nil))
}
