package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/stretchr/testify/require"
)

var (
	paramInCacheID           = 1
	token                    = "token"
	configParamsByIDTemplate = `
token: %s
proxy_url: %s
log_level: DEBUG
log_pretty_print: true
cache_methods:
- name: %s
  cache_by_params: true
  params_in_cache_id:
    - %s
`
)

func getConfig(url string, method string) (*config.Config, error) {
	template := fmt.Sprintf(configParamsByIDTemplate, token, url, method, strconv.Itoa(paramInCacheID))
	return config.NewConfig(strings.NewReader(template))
}

func TestMain(t *testing.M) {
	logger.InitDefaultLogger()
	os.Exit(t.Run())
}

func TestRpcResponsesUnmarshal(t *testing.T) {
	data := `{
		"jsonrpc": "2.0",
		"method": "test",
		"id": 5,
		"params": ["1", 2, null]
	}
	`
	request := rpcRequest{}
	err := json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	params := request.Params.([]interface{})
	require.Len(t, params, 3)

	data = `{
		"jsonrpc": "2.0",
		"method": "test",
		"id": 5,
		"params": ["1", "2"]
	}
	`
	request = rpcRequest{}
	err = json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	params = request.Params.([]interface{})
	require.Len(t, params, 2)

	data = `{
		"jsonrpc": "2.0",
		"method": "test",
		"id": 5,
		"params": {"a": "1", "b": "2"}
	}
	`
	request = rpcRequest{}
	err = json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	paramsMap := request.Params.(map[string]interface{})
	require.Len(t, paramsMap, 2)
}

func TestRpcResponsesCacheKey(t *testing.T) {
	data := `{
		"jsonrpc": "2.0",
		"method": "test",
		"id": 5,
		"params": ["1", 2, null]
	}
	`
	request := rpcRequest{}
	err := json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	params := request.Params.([]interface{})
	require.Len(t, params, 3)

	matcherImp := newMatcher()
	matcherImp.methods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheID:   []int{0, 2},
		paramsInCacheName: nil,
	}

	key1 := matcherImp.key(request)
	parts := strings.Split(key1, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)

	key2 := matcherImp.key(request)
	require.Equal(t, key1, key2)

}

func TestTransportWithCache(t *testing.T) {

	method := "test"
	requestID := "1"
	result := float64(15)

	response := rpcResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}

	responseJson, err := json.Marshal(response)
	require.NoError(t, err)
	request := rpcRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}

	jsonRequest, err := json.Marshal(request)
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, string(responseJson))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := getConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := NewServer(conf)
	require.NoError(t, err)

	frontend := httptest.NewServer(http.HandlerFunc(server.RPCProxy))
	defer frontend.Close()
	//frontendClient := frontend.Client()

	resp, err := http.Post(
		frontend.URL,
		"application/json",
		ioutil.NopCloser(bytes.NewBuffer(jsonRequest)),
	)
	require.NoError(t, err)

	responses, err := parseResponses(resp)
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, responses[0].Result, result)
	require.Equal(t, responses[0].ID, requestID)

	cache, err := server.transport.getResponseCache(request)
	require.NoError(t, err)
	require.Equal(t, cache.Result, result)
	require.Equal(t, cache.ID, requestID)

}

func TestTransportBulkRequest(t *testing.T) {

	method := "test"
	requestID1 := "1"
	requestID2 := "2"
	result1 := float64(15)
	result2 := float64(16)

	response1 := rpcResponse{
		JSONRPC: "2.0",
		ID:      requestID1,
		Result:  result1,
		Error:   nil,
	}
	response2 := rpcResponse{
		JSONRPC: "2.0",
		ID:      requestID2,
		Result:  result2,
		Error:   nil,
	}

	responseJson, err := json.Marshal(response2)
	require.NoError(t, err)

	request1 := rpcRequest{
		JSONRPC: "2.0",
		ID:      requestID1,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}
	request2 := rpcRequest{
		JSONRPC: "2.0",
		ID:      requestID2,
		Method:  method,
		Params:  []interface{}{"2", "3"},
	}

	jsonRequest, err := json.Marshal([]rpcRequest{request1, request2})
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		requests, err := parseRequests(r)
		require.NoError(t, err)
		require.Len(t, requests, 1)
		request := requests[0]
		require.Equal(t, request.ID, requestID2)
		_, err = fmt.Fprint(w, string(responseJson))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := getConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := NewServer(conf)
	require.NoError(t, err)

	err = server.transport.setResponseCache(request1, response1)
	require.NoError(t, err)

	frontend := httptest.NewServer(http.HandlerFunc(server.RPCProxy))
	defer frontend.Close()
	//frontendClient := frontend.Client()

	resp, err := http.Post(
		frontend.URL,
		"application/json",
		ioutil.NopCloser(bytes.NewBuffer(jsonRequest)),
	)
	require.NoError(t, err)

	responses, err := parseResponses(resp)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.Equal(t, responses[0].Result, result1)
	require.Equal(t, responses[0].ID, requestID1)
	require.Equal(t, responses[1].Result, result2)
	require.Equal(t, responses[1].ID, requestID2)

}
