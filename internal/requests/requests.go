package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/protofire/filecoin-rpc-proxy/internal/utils"
)

const (
	jsonRPCInvalidParams = -32602
	jsonRPCInternal      = -32603
)

type RPCResponses []RPCResponse
type RPCRequests []RPCRequest

func (r RPCRequests) FindByID(id interface{}) (RPCRequest, bool) {
	for _, req := range r {
		if utils.Equal(req.ID, id) {
			return req, true
		}
	}
	return RPCRequest{}, false
}

func (r RPCRequests) FindByPositions(ids ...int) RPCRequests {
	var res RPCRequests
	for _, idx := range ids {
		if idx < len(r) && idx >= 0 {
			res = append(res, r[idx])
		}
	}
	return res
}

func (r RPCRequests) IsEmpty() bool {
	return len(r) == 0
}

func (r RPCRequests) Methods() []string {
	methods := make([]string, len(r))
	for i := range r {
		methods[i] = r[i].Method
	}
	return methods
}

// SplitEmptyResponsePositions splits responses on non-empty / empty subsets by position
func (r RPCResponses) SplitEmptyResponsePositions() ([]int, []int) {
	var empty []int
	var nonEmpty []int
	for idx, response := range r {
		if response.IsEmpty() {
			empty = append(empty, idx)
		} else {
			nonEmpty = append(nonEmpty, idx)
		}
	}
	return nonEmpty, empty
}

func (r RPCResponses) Response() (*http.Response, error) {
	switch len(r) {
	case 0:
		return JSONRPCResponse(200, nil)
	case 1:
		return JSONRPCResponse(200, r[0])
	default:
		return JSONRPCResponse(200, r)
	}
}

type errResponse struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Error   rpcError    `json:"error"`
}

type RPCRequest struct {
	remoteAddr string
	JSONRPC    string      `json:"jsonrpc" bson:"jsonrpc"`
	ID         interface{} `json:"id,omitempty" bson:"id,omitempty"`
	Method     string      `json:"method" bson:"method"`
	Params     interface{} `json:"params,omitempty" bson:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc" bson:"jsonrpc"`
	ID      interface{} `json:"id,omitempty" bson:"id,omitempty"`
	Result  interface{} `json:"result,omitempty" bson:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty" bson:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *rpcError) Error() string {
	return fmt.Sprintf("RCP error. Code: %d. Message: %s. data: %v", r.Code, r.Message, r.Data)
}

func (r RPCResponse) IsEmpty() bool {
	return r.JSONRPC == ""
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

func DebugRequest(request *http.Request, log *logrus.Entry) {
	dump, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		log.Error(err)
	} else {
		log.Logger.SetOutput(os.Stderr)
		log.Debug(string(dump))
		log.Logger.SetOutput(os.Stdout)
	}
}

func DebugResponse(response *http.Response, log *logrus.Entry) {
	dump, err := httputil.DumpResponse(response, true)
	if err != nil {
		log.Error(err)
	} else {
		log.Logger.SetOutput(os.Stderr)
		log.Debug(string(dump))
		log.Logger.SetOutput(os.Stdout)
	}
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

func parseRequestBody(body []byte) ([]RPCRequest, error) {
	if isBatch(body) {
		var arr []RPCRequest
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch request: %w", err)
		}
		return arr, nil
	}
	var rpc RPCRequest
	if err := json.Unmarshal(body, &rpc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON request: %v", err)
	}
	return []RPCRequest{rpc}, nil
}

func parseResponseBody(body []byte) ([]RPCResponse, error) {
	if isBatch(body) {
		var arr []RPCResponse
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch response: %w", err)
		}
		return arr, nil
	}
	var rpc RPCResponse
	if err := json.Unmarshal(body, &rpc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}
	return []RPCResponse{rpc}, nil
}

func ParseRequests(req *http.Request) (RPCRequests, error) {
	var err error
	var res []RPCRequest
	body, err := utils.Read(req.Body)
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
		res = append(res, RPCRequest{
			Method: req.URL.Path,
		})
	}
	for idx := range res {
		res[idx].remoteAddr = ip
	}
	return res, nil
}

func ParseResponses(req *http.Response) (RPCResponses, []byte, error) {
	var err error
	var res RPCResponses
	body, err := utils.Read(req.Body)
	if err != nil {
		return nil, nil, err
	}
	if len(body) > 0 {
		if res, err = parseResponseBody(body); err != nil {
			return nil, nil, err
		}
	}
	return res, body, nil
}

func jsonRPCError(id interface{}, jsonCode int, msg string) interface{} {
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

func JSONRPCUnauthenticated() interface{} {
	return jsonRPCError(
		nil,
		jsonRPCInternal,
		"Unauthorized",
	)
}

func JSONInvalidResponse(message string) (*http.Response, error) {
	return JSONRPCResponse(http.StatusBadRequest, jsonRPCError(nil, jsonRPCInvalidParams, message))
}

// jsonRPCResponse returns a JSON response containing v, or a plaintext generic
// response for this httpCode and an error when JSON marshalling fails.
func JSONRPCResponse(httpCode int, v interface{}) (*http.Response, error) {
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
		Header:     map[string][]string{"Content-Type": {"application/json"}},
	}, nil
}

func JSONRPCErrorResponse(httpCode int, data []byte) (*http.Response, error) {
	rpcErr := jsonRPCError(
		nil,
		jsonRPCInternal,
		string(data),
	)
	return JSONRPCResponse(httpCode, rpcErr)
}

func Request(
	url,
	token string,
	log *logrus.Entry,
	debugHTTPRequest bool,
	debugHTTPResponse bool,
	requests RPCRequests,
) (RPCResponses, []byte, error) {
	var reqs interface{} = requests
	if len(requests) == 1 {
		reqs = requests[0]
	}
	jsonBody, err := json.Marshal(reqs)
	if err != nil {
		return nil, nil, err
	}
	body := ioutil.NopCloser(bytes.NewBuffer(jsonBody))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	if debugHTTPRequest {
		DebugRequest(req, log)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	if debugHTTPResponse {
		DebugResponse(resp, log)
	}
	if resp.StatusCode >= 300 {
		body, _ := utils.Read(resp.Body)
		return nil, nil, fmt.Errorf(
			"cannot get custom reponses. status code: %d. response: %s",
			resp.StatusCode,
			string(body),
		)
	}
	return ParseResponses(resp)
}
