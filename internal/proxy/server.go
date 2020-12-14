package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/sirupsen/logrus"
)

type Server struct {
	host   string
	port   int
	target *url.URL
	logger *logrus.Entry
	proxy  *httputil.ReverseProxy
	*transport
}

func FromConfig(ctx context.Context, c *config.Config) (*Server, error) {
	proxyURL, err := url.Parse(c.ProxyURL)
	if err != nil {
		return nil, err
	}
	log := logger.InitLogger(c.LogLevel, c.LogPrettyPrint)
	cacheImpl, err := cache.FromConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	cacher := NewResponseCache(
		cacheImpl,
		matcher.FromConfig(c),
	)
	transport := NewTransport(cacher, log, c.DebugHTTPRequest, c.DebugHTTPResponse)
	return newServer(proxyURL, c.Host, c.Port, log, transport)
}

func newServer(proxyURL *url.URL, host string, port int, log *logrus.Entry, transport *transport) (*Server, error) {
	log.Infof("Initializing proxy server for %s...", proxyURL)
	hostProxyURL := *proxyURL
	hostProxyURL.Path = ""
	transport.proxyURL = proxyURL
	s := &Server{
		host:      host,
		port:      port,
		target:    proxyURL,
		logger:    log,
		proxy:     httputil.NewSingleHostReverseProxy(&hostProxyURL),
		transport: transport,
	}
	s.proxy.Transport = transport
	return s, nil
}

func FromConfigWithTransport(c *config.Config, log *logrus.Entry, transport *transport) (*Server, error) {
	proxyURL, err := url.Parse(c.ProxyURL)
	if err != nil {
		return nil, err
	}
	return newServer(proxyURL, c.Host, c.Port, log, transport)
}

func (p *Server) RPCProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-rpc-proxy", "rpc-proxy")
	p.proxy.ServeHTTP(w, r)
}

// HealthFunc health checking
func (p *Server) HealthFunc(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		p.logger.Errorf("response send error %v", err)
	}
}

// ReadyFunc readiness checking
func (p *Server) ReadyFunc(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		p.logger.Errorf("response send error %v", err)
	}
}

// StartHTTPServer starts http server
func (p *Server) StartHTTPServer(h http.Handler) *http.Server {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", p.host, p.port),
		Handler: h,
	}

	go func() {
		p.logger.Infof("Listening on %s:%d", p.host, p.port)
		if err := server.ListenAndServe(); err != nil {
			p.logger.Infof("Listening status: %v", err)
		}
	}()

	return server
}
