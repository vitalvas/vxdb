package main

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var vxdbHttpRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "vxdb_http_requests_total",
		Help: "Number of requests",
	},
	[]string{"method"},
)
var vxdbHttpBucketRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "vxdb_http_bucket_requests_total",
		Help: "Number of requests",
	},
	[]string{"bucket", "method"},
)

func init() {
	prometheus.Register(vxdbHttpRequests)
	prometheus.Register(vxdbHttpBucketRequests)
}

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		skip := false

		for key := range reservedKeys {
			if strings.HasPrefix(r.RequestURI, "/"+key) {
				skip = true
			}
		}

		if !skip {
			vxdbHttpRequests.WithLabelValues(r.Method).Inc()
			bucket := mux.Vars(r)["bucket"]
			if bucket != "" {
				vxdbHttpBucketRequests.WithLabelValues(bucket, r.Method).Inc()
			}
		}
	})
}
