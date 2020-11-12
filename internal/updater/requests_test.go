package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/testhelpers"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {

	method := "test"
	requestID := "1"
	result := float64(15)

	response := requests.RpcResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}

	responseJson, err := json.Marshal(response)
	require.NoError(t, err)
	request := requests.RpcRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, string(responseJson))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := proxy.FromConfig(conf)
	require.NoError(t, err)

	handler := proxy.PrepareRoutes(conf, logger.Log, server)
	frontend := httptest.NewServer(handler)
	defer frontend.Close()

	token, err := auth.NewJWT(conf.JWTSecret, conf.JWTAlgorithm, []string{"admin"})
	require.NoError(t, err)

	responses, err := requests.Request(
		frontend.URL,
		string(token),
		requests.RpcRequests{request},
	)
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, responses[0].Result, result)
	require.Equal(t, responses[0].ID, requestID)

}
