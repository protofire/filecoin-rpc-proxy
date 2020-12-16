package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/ory/dockertest/v3/docker"

	"github.com/ory/dockertest/v3"

	"go.uber.org/goleak"

	"github.com/protofire/filecoin-rpc-proxy/internal/testhelpers"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/stretchr/testify/require"
)

const (
	method = "test"
	host   = "127.0.0.1"
	port   = "6379"
)

var redisURI = fmt.Sprintf("redis://%s:%s", host, port)

func TestMain(m *testing.M) { // nolint
	logger.InitDefaultLogger()
	pool, err := dockertest.NewPool("")
	if err != nil {
		logger.Log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   "redis",
		Tag:          "latest",
		ExposedPorts: []string{port},
		PortBindings: map[docker.Port][]docker.PortBinding{
			port: {{HostIP: host, HostPort: port}},
		},
	})

	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	ctx := context.Background()
	if err = pool.Retry(func() error {
		var err error
		c, err := cache.NewRedisClient(ctx, config.RedisCacheSettings{
			URI: redisURI,
		})
		if err == nil {
			_ = c.Close()
		}
		return err
	}); err != nil {
		log.Fatalf("Could not connect to redis: %s", err)
	}

	exitCode := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	if exitCode == 0 {
		if err := goleak.Find(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: Errors on successful test run: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)

}

func TestRPCResponsesUnmarshal(t *testing.T) {
	data := `{
		"jsonrpc": "2.0",
		"method": "test",
		"id": 5,
		"params": ["1", 2, null]
	}
	`
	request := requests.RPCRequest{}
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
	request = requests.RPCRequest{}
	err = json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	params = request.Params.([]interface{})
	require.Len(t, params, 2)

	data = `{
		"jsonrpc": "2.0",
		"testMethod": "test",
		"id": 5,
		"params": {"a": "1", "b": "2"}
	}
	`
	request = requests.RPCRequest{}
	err = json.Unmarshal([]byte(data), &request)
	require.NoError(t, err)
	paramsMap := request.Params.(map[string]interface{})
	require.Len(t, paramsMap, 2)
}

func TestTransportWithCache(t *testing.T) {
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

	jsonRequest, err := json.Marshal(request)
	require.NoError(t, err)

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
	require.Len(t, responses, 1)
	require.Equal(t, responses[0].Result, result)
	require.Equal(t, responses[0].ID, requestID)

	cacheResult, err := server.transport.cacher.GetResponseCache(request)
	require.NoError(t, err)
	require.Equal(t, cacheResult.Result, result)
	require.Equal(t, cacheResult.ID, requestID)
}

func TestTransportWithRedisCache(t *testing.T) {
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

	jsonRequest, err := json.Marshal(request)
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetRedisConfig(backend.URL, redisURI, method)
	require.NoError(t, err)
	ctx, done := context.WithCancel(context.Background())
	server, err := FromConfig(ctx, conf)
	require.NoError(t, err)

	defer func() {
		done()
		if err := server.Close(); err != nil {
			logger.Log.Error(err)
		}
	}()

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
	require.Len(t, responses, 1)
	require.Equal(t, responses[0].Result, result)
	require.Equal(t, responses[0].ID, requestID)

	cacheResult, err := server.transport.cacher.GetResponseCache(request)
	require.NoError(t, err)
	require.NotNil(t, cacheResult)
	require.Equal(t, cacheResult.Result, result)
	require.Equal(t, cacheResult.ID, requestID)
}

func TestTransportBulkRequest(t *testing.T) {
	requestID1 := "10"
	requestID2 := "20"
	result1 := float64(15)
	result2 := float64(16)
	response1 := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID1,
		Result:  result1,
		Error:   nil,
	}
	response2 := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID2,
		Result:  result2,
		Error:   nil,
	}

	responseJSON, err := json.Marshal(response2)
	require.NoError(t, err)

	request1 := requests.RPCRequest{
		JSONRPC: "2.0",
		ID:      requestID1,
		Method:  method,
		Params:  []interface{}{"1", "2"},
	}
	request2 := requests.RPCRequest{
		JSONRPC: "2.0",
		ID:      requestID2,
		Method:  method,
		Params:  []interface{}{"2", "3"},
	}

	jsonRequest, err := json.Marshal([]requests.RPCRequest{request1, request2})
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		reqs, err := requests.ParseRequests(r)
		require.NoError(t, err)
		require.Len(t, reqs, 1)
		request := reqs[0]
		require.Equal(t, request.ID, requestID2)
		_, err = fmt.Fprint(w, string(responseJSON))
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

	err = server.transport.cacher.SetResponseCache(request1, response1)
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
	require.Len(t, responses, 2)
	require.Equal(t, responses[0].ID, requestID1)
	require.Equal(t, responses[1].ID, requestID2)
	require.Equal(t, responses[0].Result, result1)
	require.Equal(t, responses[1].Result, result2)
}

func TestTransportBulkRequestReverseResponses(t *testing.T) {
	methods := []string{"test1", "test2", "test3", "test4", "test5"}
	var resps requests.RPCResponses
	var reqs requests.RPCRequests
	for idx, method := range methods {
		id := strconv.Itoa(idx + 1)
		reqs = append(reqs, requests.RPCRequest{
			JSONRPC: "2.0",
			ID:      id,
			Method:  method,
			Params:  []string{"1"},
		})
		resps = append(resps, requests.RPCResponse{
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

	responsesJSON, err := json.Marshal(resps)
	require.NoError(t, err)

	jsonRequest, err := json.Marshal(reqs)
	require.NoError(t, err)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprint(w, string(responsesJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfig(backend.URL, methods...)
	require.NoError(t, err)
	ctx := context.Background()
	server, err := FromConfig(ctx, conf)
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
		resp, err := server.transport.cacher.GetResponseCache(req)
		require.NoError(t, err)
		require.Equal(t, resp.ID, req.ID)
	}
}
