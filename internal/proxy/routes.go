package proxy

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/protofire/filecoin-rpc-proxy/internal/requests"
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
		r.Use(Authenticator)
		r.HandleFunc("/*", server.RPCProxy)
	})
	return r
}

func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())

		if err != nil {
			resp := requests.JsonRPCUnauthenticated()
			data, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, http.StatusText(401), 401)
				return
			}
			http.Error(w, string(data), 401)
			return
		}

		if token == nil || !token.Valid {
			resp := requests.JsonRPCUnauthenticated()
			data, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, http.StatusText(401), 401)
				return
			}
			http.Error(w, string(data), 401)
			return
		}

		// Token is authenticated, pass it through
		next.ServeHTTP(w, r)
	})
}
