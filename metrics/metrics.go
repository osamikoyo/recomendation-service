package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var RequestTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "request_total",
		Help: "Total number of requests",
	},
	[]string{"func"},
)

var RequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "request_duration",
		Help:    "Duration to route request",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"func"},
)

func InitMetrics() {
	sync.OnceFunc(func() {
		prometheus.MustRegister(RequestTotal, RequestDuration)
	})
}
