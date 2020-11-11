package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
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

func SetRequestDuration(n int64) {
	proxyRequestDuration.Set(float64(n))
}

func SetRequestCounter() {
	proxyRequests.Inc()
}
func SetRequestErrorCounter() {
	errorProxyRequests.Inc()
}
func SetRequestCachedCounter() {
	cachedProxyRequests.Inc()
}

func Register() {
	prometheus.MustRegister(proxyRequestDuration)
	prometheus.MustRegister(errorProxyRequests)
	prometheus.MustRegister(cachedProxyRequests)
	prometheus.MustRegister(proxyRequests)
}
