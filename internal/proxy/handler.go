package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"
	"github.com/sirupsen/logrus"

	"github.com/go-chi/chi/middleware"
)

type transport struct {
	logger            *logrus.Entry
	cacher            ResponseCacher
	proxyURL          *url.URL
	debugHTTPRequest  bool
	debugHTTPResponse bool
}

// nolint
func NewTransport(cacher ResponseCacher, logger *logrus.Entry, debugHTTPRequest, debugHttpResponse bool) *transport {
	return &transport{
		logger:            logger,
		cacher:            cacher,
		debugHTTPRequest:  debugHTTPRequest,
		debugHTTPResponse: debugHttpResponse,
	}
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
	methods := parsedRequests.Methods()
	log = log.WithField("methods", methods)
	for _, method := range methods {
		metrics.SetRequestsCounterByMethod(method)
	}

	preparedResponses, err := t.fromCache(parsedRequests)
	if err != nil {
		log.Errorf("Cannot build prepared responses: %v", err)
		preparedResponses = make(requests.RPCResponses, len(parsedRequests))
	}

	cachedRequestIdx, proxyRequestIdx := preparedResponses.SplitEmptyResponsePositions()

	// build requests to proxy
	proxyRequests := parsedRequests.FindByPositions(proxyRequestIdx...)
	cachedRequests := parsedRequests.FindByPositions(cachedRequestIdx...)
	cachedMethods := cachedRequests.Methods()

	if len(cachedRequests) > 0 {
		metrics.SetRequestsCachedCounterByMethods(cachedMethods...)
	}

	var proxyBody []byte
	switch len(proxyRequests) {
	case 0:
		log.Debug("returning proxy response...")
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
	req.ContentLength = int64(len(proxyBody))
	req.Host = t.proxyURL.Host
	log.Debug("Forwarding request...")
	if t.debugHTTPRequest {
		requests.DebugRequest(req, log)
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	elapsed := time.Since(start)
	metrics.SetRequestDuration(elapsed.Milliseconds())
	if err != nil {
		metrics.SetRequestsErrorCounterByMethods(methods...)
		return res, err
	}
	if t.debugHTTPResponse {
		requests.DebugResponse(res, log)
	}
	// no need cache. Return without parsing response
	if !t.isCacheableRequests(parsedRequests) && len(cachedMethods) == 0 {
		return res, nil
	}
	responses, body, err := requests.ParseResponses(res)
	if err != nil {
		metrics.SetRequestsErrorCounterByMethods(methods...)
		return requests.JSONRPCErrorResponse(res.StatusCode, body)
	}

	for idx, response := range responses {
		if response.Error == nil {
			if request, ok := parsedRequests.FindByID(response.ID); ok {
				if t.cacher.Matcher().IsCacheable(request.Method) {
					if err := t.cacher.SetResponseCache(request, response); err != nil {
						t.logger.Errorf("Cannot set cached response: %v", err)
					}
				}
			}
		}
		preparedResponses[proxyRequestIdx[idx]] = response
	}

	resp, err := preparedResponses.Response()
	if err != nil {
		t.logger.Errorf("Cannot prepare response from cached responses: %v", err)
		return resp, err
	}
	return resp, nil
}

func (t *transport) isCacheableRequests(reqs requests.RPCRequests) bool {
	for _, req := range reqs {
		if !t.cacher.Matcher().IsCacheable(req.Method) {
			return false
		}
	}
	return true
}

// fromCache checks presence of messages in the cache
func (t *transport) fromCache(reqs requests.RPCRequests) (requests.RPCResponses, error) {
	results := make(requests.RPCResponses, len(reqs))
	for idx, request := range reqs {
		response, err := t.cacher.GetResponseCache(request)
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

func (t *transport) Close() error {
	return t.cacher.Cacher().Close()
}
