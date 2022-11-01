package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	rpcDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "rpc_durations_seconds",
		Help: "RPC latency distributions.",
	})

	observedValues = []float64{0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 1}
)

func init() {
	prometheus.MustRegister(rpcDurationsHistogram)
}

func main() {

	for _, val := range observedValues {
		rpcDurationsHistogram.Observe(val)
	}

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(":8888", nil))
}
