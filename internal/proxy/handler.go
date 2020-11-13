package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"
	"github.com/sirupsen/logrus"

	"github.com/go-chi/chi/middleware"
)

type transport struct {
	logger  *logrus.Entry
	cache   cache.Cache
	matcher matcher.Matcher
}

// nolint
func NewTransport(cache cache.Cache, matcher matcher.Matcher, logger *logrus.Entry) *transport {
	return &transport{
		logger:  logger,
		cache:   cache,
		matcher: matcher,
	}
}

func (t *transport) setResponseCache(req requests.RPCRequest, resp requests.RPCResponse) error {
	key := t.matcher.Key(req.Method, req.Params)
	if key == "" {
		return nil
	}
	return t.cache.Set(key, resp)
}

func (t *transport) getResponseCache(req requests.RPCRequest) (requests.RPCResponse, error) {
	resp := requests.RPCResponse{}
	key := t.matcher.Key(req.Method, req.Params)
	if key == "" {
		return resp, nil
	}
	data, err := t.cache.Get(key)
	if err != nil {
		return resp, err
	}
	if data == nil {
		return resp, nil
	}
	resp, ok := data.(requests.RPCResponse)
	if ok {
		return resp, nil
	}
	err = json.Unmarshal(data.([]byte), &resp)
	return resp, err
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	metrics.SetRequestsCounter()
	log := t.logger
	if reqID := middleware.GetReqID(req.Context()); reqID != "" {
		log = log.WithField("requestID", reqID)
	}
	start := time.Now()

	parsedRequests, err := requests.ParseRequests(req)
	if err != nil {
		log.Errorf("Failed to parse requests: %v", err)
		metrics.SetRequestsErrorCounter()
		resp, err := requests.JSONInvalidResponse(err.Error())
		if err != nil {
			log.Errorf("Failed to prepare error response: %v", err)
			return nil, err
		}
		return resp, nil
	}
	log = log.WithField("methods", parsedRequests.Methods())

	preparedResponses, err := t.fromCache(parsedRequests)
	if err != nil {
		log.Errorf("Cannot build prepared responses: %v", err)
		preparedResponses = make(requests.RPCResponses, len(parsedRequests))
	}

	proxyRequestIdx := preparedResponses.BlankResponses()
	// build requests to proxy
	var proxyRequests requests.RPCRequests
	for _, idx := range proxyRequestIdx {
		proxyRequests = append(proxyRequests, parsedRequests[idx])
	}

	inCacheRequestsCount := len(parsedRequests) - len(proxyRequests)

	var proxyBody []byte
	switch len(proxyRequests) {
	case 0:
		metrics.SetRequestsCachedCounter(inCacheRequestsCount)
		return preparedResponses.Response()
	case 1:
		proxyBody, err = json.Marshal(proxyRequests[0])
	default:
		proxyBody, err = json.Marshal(proxyRequests)
	}
	if err != nil {
		log.Errorf("Failed to construct invalid cacheParams response: %v", err)
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(proxyBody))
	log.Debug("Forwarding request...")
	req.Host = req.RemoteAddr
	res, err := http.DefaultTransport.RoundTrip(req)
	elapsed := time.Since(start)
	metrics.SetRequestDuration(elapsed.Milliseconds())
	if err != nil {
		metrics.SetRequestsErrorCounter()
		return res, err
	}
	responses, body, err := requests.ParseResponses(res)
	if err != nil {
		metrics.SetRequestsErrorCounter()
		return requests.JSONRPCErrorResponse(res.StatusCode, body)
	}

	for idx, response := range responses {
		if response.Error == nil {
			request, ok := parsedRequests.FindByID(response.ID)
			if !ok {
				request = parsedRequests[proxyRequestIdx[idx]]
			}
			err := t.setResponseCache(request, response)
			if err != nil {
				t.logger.Errorf("Cannot set cached response: %v", err)
			}
		}
		preparedResponses[proxyRequestIdx[idx]] = response
	}

	metrics.SetRequestsCachedCounter(inCacheRequestsCount)

	resp, err := preparedResponses.Response()
	if err != nil {
		t.logger.Errorf("Cannot prepare response from cached responses: %v", err)
		return resp, err
	}
	return resp, nil
}

// fromCache checks presence of messages in the cache
func (t *transport) fromCache(reqs requests.RPCRequests) (requests.RPCResponses, error) {
	results := make(requests.RPCResponses, len(reqs))
	for idx, request := range reqs {
		response, err := t.getResponseCache(request)
		if err != nil {
			cacheErr := &cache.Error{}
			if errors.As(err, cacheErr) {
				t.logger.Errorf("Cannot get cache value for testMethod %q: %v", request.Method, cacheErr)
			} else {
				return results, err
			}
		}
		response.ID = request.ID
		results[idx] = response
	}
	return results, nil
}
