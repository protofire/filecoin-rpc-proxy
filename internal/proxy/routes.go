package proxy

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/sirupsen/logrus"
)

func PrepareRoutes(c *config.Config, log *logrus.Entry, server *Server) *chi.Mux {
	tokenAuth := auth.JWTSecret(c.JWTSecret, c.JWTAlgorithm)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logger.NewStructuredLogger(log.Logger))
	r.Use(middleware.Recoverer)
	r.HandleFunc("/healthz", server.HealthFunc)
	r.HandleFunc("/ready", server.ReadyFunc)
	r.Handle("/metrics", promhttp.Handler())
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)
		r.HandleFunc("/*", server.RPCProxy)
	})
	return r
}
