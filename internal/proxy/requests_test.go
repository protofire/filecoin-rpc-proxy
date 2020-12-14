package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/auth"
	"github.com/protofire/filecoin-rpc-proxy/internal/testhelpers"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	method := "test"
	requestID := "1"
	result := float64(15)

	response := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}

	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)
	request := requests.RPCRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfig(backend.URL, method)
	require.NoError(t, err)
	ctx := context.Background()
	server, err := FromConfig(ctx, conf)
	require.NoError(t, err)

	handler := PrepareRoutes(conf, logger.Log, server)
	frontend := httptest.NewServer(handler)
	defer frontend.Close()

	token, err := auth.NewJWT(conf.JWT(), conf.JWTAlgorithm, conf.JWTPermissions)
	require.NoError(t, err)

	responses, _, err := requests.Request(
		frontend.URL,
		string(token),
		logger.Log,
		false,
		false,
		requests.RPCRequests{request},
	)
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, responses[0].Result, result)
	require.Equal(t, responses[0].ID, requestID)
}
