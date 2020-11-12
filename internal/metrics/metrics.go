package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "cache",
		Name:      "size",
		Help:      "The proxy cache size",
	})
	proxyRequestDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "proxy",
		Name:      "request_duration",
		Help:      "The proxy request duration",
	})
	proxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests",
		Help:      "The total number of processed proxy requests",
	})
	cachedProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_cached",
		Help:      "The total number of cached proxy requests",
	})
	errorProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_error",
		Help:      "The total number of failed proxy requests",
	})
)

// SetRequestDuration ...
func SetRequestDuration(n int64) {
	proxyRequestDuration.Observe(float64(n))
}

// SetCacheSize ...
func SetCacheSize(n int64) {
	cacheSize.Set(float64(n))
}

// SetRequestsCounter ...
func SetRequestsCounter() {
	proxyRequests.Inc()
}

// SetRequestsErrorCounter ...
func SetRequestsErrorCounter() {
	errorProxyRequests.Inc()
}

// SetRequestsCachedCounter ...
func SetRequestsCachedCounter(n int) {
	cachedProxyRequests.Add(float64(n))
}

// Register ...
func Register() {
	prometheus.MustRegister(proxyRequestDuration)
	prometheus.MustRegister(errorProxyRequests)
	prometheus.MustRegister(cachedProxyRequests)
	prometheus.MustRegister(proxyRequests)
}
