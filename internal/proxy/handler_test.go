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
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/testhelpers"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/stretchr/testify/require"
)

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
	request := requests.RpcRequest{}
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
	request = requests.RpcRequest{}
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
	request = requests.RpcRequest{}
	err = json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	paramsMap := request.Params.(map[string]interface{})
	require.Len(t, paramsMap, 2)
}

func TestTransportWithCache(t *testing.T) {

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

	conf, err := testhelpers.GetConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := FromConfig(conf)
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

	responses, _, err := requests.ParseResponses(resp)
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
	requestID1 := "10"
	requestID2 := "20"
	result1 := float64(15)
	result2 := float64(16)

	response1 := requests.RpcResponse{
		JSONRPC: "2.0",
		ID:      requestID1,
		Result:  result1,
		Error:   nil,
	}
	response2 := requests.RpcResponse{
		JSONRPC: "2.0",
		ID:      requestID2,
		Result:  result2,
		Error:   nil,
	}

	responseJson, err := json.Marshal(response2)
	require.NoError(t, err)

	request1 := requests.RpcRequest{
		JSONRPC: "2.0",
		ID:      requestID1,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}
	request2 := requests.RpcRequest{
		JSONRPC: "2.0",
		ID:      requestID2,
		Method:  method,
		Params:  []interface{}{"2", "3"},
	}

	jsonRequest, err := json.Marshal([]requests.RpcRequest{request1, request2})
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		reqs, err := requests.ParseRequests(r)
		require.NoError(t, err)
		require.Len(t, reqs, 1)
		request := reqs[0]
		require.Equal(t, request.ID, requestID2)
		_, err = fmt.Fprint(w, string(responseJson))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfig(backend.URL, method)
	require.NoError(t, err)
	server, err := FromConfig(conf)
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

	responses, _, err := requests.ParseResponses(resp)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.Equal(t, responses[0].ID, requestID1)
	require.Equal(t, responses[1].ID, requestID2)
	require.Equal(t, responses[0].Result, result1)
	require.Equal(t, responses[1].Result, result2)

}

func TestTransportBulkRequestReverseResponses(t *testing.T) {

	methods := []string{"test1", "test2", "test3", "test4", "test5"}

	var resps requests.RpcResponses
	var reqs requests.RpcRequests

	for idx, method := range methods {
		id := strconv.Itoa(idx + 1)
		reqs = append(reqs, requests.RpcRequest{
			JSONRPC: "2.0",
			ID:      id,
			Method:  method,
			Params:  []string{"1"},
		})
		resps = append(resps, requests.RpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  id,
			Error:   nil,
		})
	}

	for left, right := 0, len(resps)-1; left < right; {
		resps[left], resps[right] = resps[right], resps[left]
		left++
		right--
	}
	require.Equal(t, strconv.Itoa(len(methods)), resps[0].ID)

	responsesJson, err := json.Marshal(resps)
	require.NoError(t, err)

	jsonRequest, err := json.Marshal(reqs)
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprint(w, string(responsesJson))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfig(backend.URL, methods...)
	require.NoError(t, err)
	server, err := FromConfig(conf)
	require.NoError(t, err)

	frontend := httptest.NewServer(http.HandlerFunc(server.RPCProxy))
	defer frontend.Close()

	resp, err := http.Post(
		frontend.URL,
		"application/json",
		ioutil.NopCloser(bytes.NewBuffer(jsonRequest)),
	)
	require.NoError(t, err)

	responses, _, err := requests.ParseResponses(resp)
	require.NoError(t, err)
	require.Len(t, responses, len(methods))

	for _, req := range reqs {
		resp, err := server.transport.getResponseCache(req)
		require.NoError(t, err)
		require.Equal(t, resp.ID, req.ID)
	}

}
