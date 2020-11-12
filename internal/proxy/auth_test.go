package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/stretchr/testify/require"
)

func TestServerAuxiliaryFunc(t *testing.T) {

	method := "test"
	conf, err := getConfig("http://test.com", method)
	require.NoError(t, err)
	server, err := NewServer(conf)
	require.NoError(t, err)
	handler := PrepareRoutes(conf, logger.Log, server)

	s := httptest.NewServer(handler)
	defer s.Close()

	for _, path := range []string{"healthz", "ready", "metrics"} {
		t.Run(fmt.Sprintf("test_%s", path), func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/%s", s.URL, path))
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestServerJWTAuthFunc401(t *testing.T) {

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	method := "test"
	conf, err := getConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := NewServer(conf)
	require.NoError(t, err)
	handler := PrepareRoutes(conf, logger.Log, server)

	s := httptest.NewServer(handler)
	defer s.Close()

	resp, err := http.Get(fmt.Sprintf("%s/%s", s.URL, "/test"))
	require.NoError(t, err)
	require.Equal(t, 401, resp.StatusCode)

}

func TestServerJWTAuthFunc(t *testing.T) {

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	method := "test"
	conf, err := getConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := NewServer(conf)
	require.NoError(t, err)
	handler := PrepareRoutes(conf, logger.Log, server)

	frontend := httptest.NewServer(handler)
	defer frontend.Close()

	jwtToken, err := auth.NewJWT(conf.JWTSecret, "admin")
	require.NoError(t, err)
	url := fmt.Sprintf("%s/%s", frontend.URL, "test")

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

}
