package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "cache",
		Name:      "cache_size",
		Help:      "The proxy cache size",
	})
	proxyRequestDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "proxy",
		Name:      "proxy_request_duration",
		Help:      "The proxy request duration",
	})
	proxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "proxy_requests",
		Help:      "The total number of processed proxy requests",
	})
	cachedProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "proxy_requests_cached",
		Help:      "The total number of cached proxy requests",
	})
	errorProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "proxy_requests_error",
		Help:      "The total number of failed proxy requests",
	})
)

// SetRequestDuration ...
func SetRequestDuration(n int64) {
	proxyRequestDuration.Set(float64(n))
}

// SetCacheSize ...
func SetCacheSize(n int64) {
	cacheSize.Set(float64(n))
}

// SetRequestCounter ...
func SetRequestCounter() {
	proxyRequests.Inc()
}

// SetRequestErrorCounter ...
func SetRequestErrorCounter() {
	errorProxyRequests.Inc()
}

// SetRequestCachedCounter ...
func SetRequestCachedCounter() {
	cachedProxyRequests.Inc()
}

// Register ...
func Register() {
	prometheus.MustRegister(proxyRequestDuration)
	prometheus.MustRegister(errorProxyRequests)
	prometheus.MustRegister(cachedProxyRequests)
	prometheus.MustRegister(proxyRequests)
}
