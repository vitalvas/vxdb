package app

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var vxdbVersion = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "vxdb_build_info",
		Help: "VxDB build info",
	},
	[]string{"version", "commit", "date"},
)
var vxdbHTTPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "vxdb_http_requests_total",
		Help: "Number of requests",
	},
	[]string{"method"},
)
var vxdbHTTPBucketRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "vxdb_http_bucket_requests_total",
		Help: "Number of requests",
	},
	[]string{"bucket", "method"},
)
var vxdbHTTPRequestsDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "vxdb_http_requests_seconds",
		Help:    "Latency of requests in second",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method"},
)

func init() {
	prometheus.Register(vxdbVersion)
	prometheus.Register(vxdbHTTPRequests)
	prometheus.Register(vxdbHTTPBucketRequests)
	prometheus.Register(vxdbHTTPRequestsDuration)
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(vxdbHTTPRequestsDuration.WithLabelValues(r.Method))

		next.ServeHTTP(w, r)

		skip := false

		for key := range reservedKeys {
			if strings.HasPrefix(r.RequestURI, "/"+key) {
				skip = true
			}
		}

		if !skip {
			vxdbHTTPRequests.WithLabelValues(r.Method).Inc()
			bucket := mux.Vars(r)["bucket"]
			if bucket != "" {
				vxdbHTTPBucketRequests.WithLabelValues(bucket, r.Method).Inc()
			}
		}

		timer.ObserveDuration()
	})
}
