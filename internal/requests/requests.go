package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
)

const (
	jsonRPCInvalidParams = -32602
	jsonRPCInternal      = -32603
)

type RpcResponses []RpcResponse
type RpcRequests []RpcRequest

func (r RpcRequests) FindByID(id interface{}) (RpcRequest, bool) {
	for _, req := range r {
		if req.ID == id {
			return req, true
		}
	}
	return RpcRequest{}, false
}

func (r RpcRequests) IsEmpty() bool {
	return len(r) == 0
}

func (r RpcRequests) Methods() []string {
	methods := make([]string, len(r))
	for i := range r {
		methods[i] = r[i].Method
	}
	return methods
}

func (r RpcResponses) BlankResponses() []int {
	var results []int
	for idx, response := range r {
		if !response.initialized() {
			results = append(results, idx)
		}
	}
	return results
}

func (r RpcResponses) Response() (*http.Response, error) {
	switch len(r) {
	case 0:
		return JsonRPCResponse(200, nil)
	case 1:
		return JsonRPCResponse(200, r[0])
	default:
		return JsonRPCResponse(200, r)
	}
}

type errResponse struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Error   rpcError    `json:"error"`
}

type RpcRequest struct {
	remoteAddr string
	JSONRPC    string      `json:"jsonrpc"`
	ID         interface{} `json:"id,omitempty"`
	Method     string      `json:"method"`
	Params     interface{} `json:"params,omitempty"`
}

type RpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (r RpcResponse) initialized() bool {
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

func parseRequestBody(body []byte) ([]RpcRequest, error) {
	if isBatch(body) {
		var arr []RpcRequest
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch request: %w", err)
		}
		return arr, nil
	} else {
		var rpc RpcRequest
		if err := json.Unmarshal(body, &rpc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON request: %v", err)
		}
		return []RpcRequest{rpc}, nil
	}
}

func parseResponseBody(body []byte) ([]RpcResponse, error) {
	if isBatch(body) {
		var arr []RpcResponse
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch response: %w", err)
		}
		return arr, nil
	} else {
		var rpc RpcResponse
		if err := json.Unmarshal(body, &rpc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %v", err)
		}
		return []RpcResponse{rpc}, nil
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

func ParseRequests(req *http.Request) (RpcRequests, error) {
	var err error
	var res []RpcRequest
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
		res = append(res, RpcRequest{
			Method: req.URL.Path,
		})
	}
	for idx := range res {
		res[idx].remoteAddr = ip
	}
	return res, nil
}

func ParseResponses(req *http.Response) (RpcResponses, error) {
	var err error
	var res RpcResponses
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

func JsonRPCError(id interface{}, jsonCode int, msg string) interface{} {
	resp := errResponse{
		Version: "2.0",
		ID:      id,
		Error: rpcError{
			Code:    jsonCode,
			Message: msg,
		},
	}
	return resp
}

func JsonRPCUnauthenticated() interface{} {
	return JsonRPCError(
		nil,
		jsonRPCInternal,
		"Unauthenticated",
	)
}

func JsonInvalidResponse(message string) (*http.Response, error) {
	return JsonRPCResponse(http.StatusBadRequest, JsonRPCError(nil, jsonRPCInvalidParams, message))
}

// jsonRPCResponse returns a JSON response containing v, or a plaintext generic
// response for this httpCode and an error when JSON marshalling fails.
func JsonRPCResponse(httpCode int, v interface{}) (*http.Response, error) {
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

func Request(url, token string, requests RpcRequests) (RpcResponses, error) {
	jsonBody, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}
	body := ioutil.NopCloser(bytes.NewBuffer(jsonBody))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	return ParseResponses(resp)
}
