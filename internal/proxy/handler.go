package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"
	"github.com/sirupsen/logrus"

	"github.com/go-chi/chi/middleware"
)

const (
	jsonRPCTimeout       = -32000
	jsonRPCUnavailable   = -32601
	jsonRPCInvalidParams = -32602
	jsonRPCInternal      = -32603
)

type rpcResponses []rpcResponse
type rpcRequests []rpcRequest

func (r rpcRequests) methods() []string {
	methods := make([]string, len(r))
	for i := range r {
		methods[i] = r[i].Method
	}
	return methods
}

func (c rpcResponses) initializedResponses() []int {
	var results []int
	for idx, response := range c {
		if response.initialized() {
			results = append(results, idx)
		}
	}
	return results
}

func (c rpcResponses) uninitializedResponses() []int {
	var results []int
	for idx, response := range c {
		if !response.initialized() {
			results = append(results, idx)
		}
	}
	return results
}

func (c rpcResponses) Response() (*http.Response, error) {
	switch len(c) {
	case 0:
		return jsonRPCResponse(200, nil)
	case 1:
		return jsonRPCResponse(200, c[0])
	default:
		return jsonRPCResponse(200, c)
	}
}

type transport struct {
	logger  *logrus.Entry
	cache   cache.Cache
	matcher matcher
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type errResponse struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   responseError   `json:"error"`
}

type rpcRequest struct {
	JSONRPC    string `json:"jsonrpc"`
	remoteAddr string
	ID         json.RawMessage   `json:"id,omitempty"`
	Method     string            `json:"method"`
	Params     []json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func (r rpcResponse) initialized() bool {
	return r.JSONRPC != ""
}

func isBatch(msg []byte) bool {
	for _, c := range msg {
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// getIP returns the original IP address from the request, checking special headers before falling back to remoteAddr.
func getIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Trim off any others: A.B.C.D[,X.X.X.X,Y.Y.Y.Y,]
		return strings.SplitN(ip, ",", 1)[0]
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

func parseRequestBody(body []byte) ([]rpcRequest, error) {
	if isBatch(body) {
		var arr []rpcRequest
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch request: %w", err)
		}
		return arr, nil
	} else {
		var rpc rpcRequest
		if err := json.Unmarshal(body, &rpc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON request: %v", err)
		}
		return []rpcRequest{rpc}, nil
	}
}

func parseResponseBody(body []byte) ([]rpcResponse, error) {
	if isBatch(body) {
		var arr []rpcResponse
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch request: %w", err)
		}
		return arr, nil
	} else {
		var rpc rpcResponse
		if err := json.Unmarshal(body, &rpc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON request: %v", err)
		}
		return []rpcResponse{rpc}, nil
	}
}

func readBody(r io.ReadCloser) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Log.Errorf("cannot close http request body: %v", err)
		}
	}()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	return body, nil
}

func parseRequests(req *http.Request) (rpcRequests, error) {
	var err error
	var res []rpcRequest
	body, err := readBody(req.Body)
	if err != nil {
		return nil, err
	}
	ip := getIP(req)
	if len(body) > 0 {
		if res, err = parseRequestBody(body); err != nil {
			return nil, err
		}
	}
	if len(res) == 0 {
		res = append(res, rpcRequest{
			Method: req.URL.Path,
		})
	}
	for idx := range res {
		res[idx].remoteAddr = ip
	}
	return res, nil
}

func parseResponses(req *http.Response) ([]rpcResponse, error) {
	var err error
	var res []rpcResponse
	body, err := readBody(req.Body)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		if res, err = parseResponseBody(body); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func jsonRPCError(id json.RawMessage, jsonCode int, msg string) interface{} {
	resp := errResponse{
		Version: "2.0",
		ID:      id,
		Error: responseError{
			Code:    jsonCode,
			Message: msg,
		},
	}
	return resp
}

func jsonRPCUnauthorized(id json.RawMessage, method string) interface{} {
	return jsonRPCError(id, jsonRPCUnavailable, fmt.Sprintf("You are not authorized to make this request: %s", method))
}

func jsonRPCLimit(id json.RawMessage) interface{} {
	return jsonRPCError(id, jsonRPCTimeout, "You hit the request limit")
}

func jsonRPCBlockRangeLimit(id json.RawMessage, blocks, limit uint64) interface{} {
	return jsonRPCError(id, jsonRPCInvalidParams, fmt.Sprintf("Requested range of blocks (%d) is larger than limit (%d).", blocks, limit))
}

// jsonRPCResponse returns a JSON response containing v, or a plaintext generic
// response for this httpCode and an error when JSON marshalling fails.
func jsonRPCResponse(httpCode int, v interface{}) (*http.Response, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return &http.Response{
			Body:       ioutil.NopCloser(strings.NewReader(http.StatusText(httpCode))),
			StatusCode: httpCode,
		}, fmt.Errorf("failed to serialize JSON: %v", err)
	}
	return &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader(body)),
		StatusCode: httpCode,
	}, nil
}

func (t *transport) setResponseCache(req rpcRequest, resp rpcResponse) error {
	key := t.matcher.key(req)
	if key == "" {
		return nil
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return t.cache.Set(key, data)
}

func (t *transport) getResponseCache(req rpcRequest) (rpcResponse, error) {
	resp := rpcResponse{}
	key := t.matcher.key(req)
	if key == "" {
		return resp, nil
	}
	data, err := t.cache.Get(key)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	metrics.SetRequestCounter()
	log := t.logger
	if reqID := middleware.GetReqID(req.Context()); reqID != "" {
		log = log.WithField("requestID", reqID)
	}
	start := time.Now()

	parsedRequests, err := parseRequests(req)
	if err != nil {
		log.Error("Failed to parse requests: %v", err)
		metrics.SetRequestErrorCounter()
		resp, err := jsonRPCResponse(http.StatusBadRequest, jsonRPCError(nil, jsonRPCInvalidParams, err.Error()))
		if err != nil {
			log.Errorf("Failed to prepare error response: %v", err)
			return nil, err
		}
		return resp, nil
	}
	log = log.WithField("methods", parsedRequests.methods())

	cachedResponses, err := t.fromCache(parsedRequests)
	if err != nil {
		log.Errorf("Cannot get cached responses: %v", err)
		cachedResponses = make(rpcResponses, len(parsedRequests))
	}

	proxyRequestIdx := cachedResponses.uninitializedResponses()
	var proxyRequests rpcRequests
	for _, idx := range proxyRequestIdx {
		proxyRequests = append(proxyRequests, parsedRequests[idx])
	}

	var proxyBody []byte
	switch len(proxyRequests) {
	case 0:
		return cachedResponses.Response()
	case 1:
		proxyBody, err = json.Marshal(proxyRequests[0])
	default:
		proxyBody, err = json.Marshal(proxyRequests)
	}
	if err != nil {
		log.Errorf("Failed to construct invalid params response: %v", err)
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(proxyBody))
	log.Debug("Forwarding request...")
	req.Host = req.RemoteAddr
	res, err := http.DefaultTransport.RoundTrip(req)
	elapsed := time.Since(start)
	metrics.SetRequestDuration(elapsed.Milliseconds())
	if err != nil {
		return res, err
	}
	responses, err := parseResponses(res)
	if err != nil {
		return res, err
	}

	for idx, response := range responses {
		if len(response.Error) == 0 {
			err := t.setResponseCache(parsedRequests[proxyRequestIdx[idx]], response)
			if err != nil {
				t.logger.Errorf("Cannot set cached response: %v", err)
			}
		}
		cachedResponses[proxyRequestIdx[idx]] = response
	}

	resp, err := cachedResponses.Response()
	if err != nil {
		t.logger.Errorf("Cannot prepare response from cached responses: %v", err)
		return resp, err
	}
	return resp, nil
}

// fromCache checks presence of messages in the cache
func (t *transport) fromCache(requests rpcRequests) (rpcResponses, error) {
	results := make(rpcResponses, len(requests))
	for idx, request := range requests {
		response, err := t.getResponseCache(request)
		if err != nil {
			cacheErr := &cache.Error{}
			if errors.As(err, cacheErr) {
				t.logger.Errorf("Cannot get cache value for method %q: %v", request.Method, cacheErr)
			} else {
				return results, err
			}
		}
		response.ID = request.ID
		results[idx] = response
	}
	return results, nil
}
