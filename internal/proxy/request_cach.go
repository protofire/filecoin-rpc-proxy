package proxy

import (
	"encoding/json"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"
	"github.com/protofire/filecoin-rpc-proxy/internal/requests"
)

type ResponseCache struct {
	cache   cache.Cache
	matcher matcher.Matcher
}

func NewResponseCache(cache cache.Cache, matcher matcher.Matcher) *ResponseCache {
	return &ResponseCache{
		cache:   cache,
		matcher: matcher,
	}
}

type ResponseCacher interface {
	SetResponseCache(requests.RPCRequest, requests.RPCResponse) error
	GetResponseCache(req requests.RPCRequest) (requests.RPCResponse, error)
	Matcher() matcher.Matcher
}

func (rc *ResponseCache) SetResponseCache(req requests.RPCRequest, resp requests.RPCResponse) error {
	key := rc.matcher.Key(req.Method, req.Params)
	if key == "" {
		return nil
	}
	return rc.cache.Set(key, resp)
}

func (rc *ResponseCache) GetResponseCache(req requests.RPCRequest) (requests.RPCResponse, error) {
	resp := requests.RPCResponse{}
	key := rc.matcher.Key(req.Method, req.Params)
	if key == "" {
		return resp, nil
	}
	data, err := rc.cache.Get(key)
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

func (rc *ResponseCache) Matcher() matcher.Matcher {
	return rc.matcher
}
