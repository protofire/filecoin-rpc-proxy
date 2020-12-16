package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels    = []string{"method"}
	cacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "proxy",
		Name:      "cache_size",
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
	proxyRequestsByMethod = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_method",
		Help:      "The total number of processed proxy requests by method",
	}, labels)
	cachedProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_cached",
		Help:      "The total number of cached proxy requests",
	})
	cachedProxyRequestsByMethod = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_method_cached",
		Help:      "The total number of cached proxy requests by method",
	}, labels)
	errorProxyRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_error",
		Help:      "The total number of failed proxy requests",
	})
	errorProxyRequestsByMethod = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxy",
		Name:      "requests_method_error",
		Help:      "The total number of failed proxy requests",
	}, labels)
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

// SetRequestsCounterByMethod ...
func SetRequestsCounterByMethod(method string) {
	proxyRequestsByMethod.With(prometheus.Labels{"method": method}).Inc()
}

// SetRequestsErrorCounter ...
func SetRequestsErrorCounter() {
	errorProxyRequests.Inc()
}

// SetRequestsErrorCounterByMethod ...
func SetRequestsErrorCounterByMethod(method string) {
	errorProxyRequestsByMethod.With(prometheus.Labels{"method": method}).Inc()
}

// SetRequestsErrorCounterByMethods ...
func SetRequestsErrorCounterByMethods(methods ...string) {
	for _, method := range methods {
		SetRequestsErrorCounterByMethod(method)
	}
	errorProxyRequests.Inc()
}

// SetRequestsCachedCounter ...
func SetRequestsCachedCounter(n int) {
	cachedProxyRequests.Add(float64(n))
}

// SetRequestsCachedCounterByMethod ...
func SetRequestsCachedCounterByMethod(method string) {
	cachedProxyRequestsByMethod.With(prometheus.Labels{"method": method}).Inc()
}

// SetRequestsCachedCounterByMethods ...
func SetRequestsCachedCounterByMethods(methods ...string) {
	SetRequestsCachedCounter(len(methods))
	for _, method := range methods {
		SetRequestsCachedCounterByMethod(method)
	}
}

// Register ...
func Register() {
	prometheus.MustRegister(proxyRequestDuration)
	prometheus.MustRegister(errorProxyRequests)
	prometheus.MustRegister(errorProxyRequestsByMethod)
	prometheus.MustRegister(cachedProxyRequests)
	prometheus.MustRegister(cachedProxyRequestsByMethod)
	prometheus.MustRegister(proxyRequests)
	prometheus.MustRegister(proxyRequestsByMethod)
}
