package proxy

import (
	"github.com/hashicorp/go-multierror"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"
	"github.com/protofire/filecoin-rpc-proxy/internal/requests"
)

// ResponseCache implements ResponseCacher interface
type ResponseCache struct {
	cache   cache.Cache
	matcher matcher.Matcher
}

// NewResponseCache fabric
func NewResponseCache(cache cache.Cache, matcher matcher.Matcher) *ResponseCache {
	return &ResponseCache{
		cache:   cache,
		matcher: matcher,
	}
}

// ResponseCacher interface
type ResponseCacher interface {
	SetResponseCache(requests.RPCRequest, requests.RPCResponse) error
	GetResponseCache(req requests.RPCRequest) (requests.RPCResponse, error)
	Matcher() matcher.Matcher
	Cacher() cache.Cache
}

// SetResponseCache sets response cache based on the request
func (rc *ResponseCache) SetResponseCache(req requests.RPCRequest, resp requests.RPCResponse) error {
	keys := rc.matcher.Keys(req.Method, req.Params)
	if len(keys) == 0 {
		return nil
	}
	mErr := &multierror.Error{}
	for _, key := range keys {
		mErr = multierror.Append(mErr, rc.cache.Set(key.Key, req, resp))
	}
	return mErr.ErrorOrNil()
}

// GetResponseCache return response from the cache for the request
func (rc *ResponseCache) GetResponseCache(req requests.RPCRequest) (requests.RPCResponse, error) {
	keys := rc.matcher.Keys(req.Method, req.Params)
	if len(keys) == 0 {
		return requests.RPCResponse{}, nil
	}
	mErr := &multierror.Error{}
	for _, key := range keys {
		resp, err := rc.cache.Get(key.Key)
		if err != nil {
			mErr = multierror.Append(mErr, err)
			continue
		}
		if resp.IsEmpty() {
			continue
		}
		return resp, nil
	}
	return requests.RPCResponse{}, nil
}

// Matcher interface implementation
func (rc *ResponseCache) Matcher() matcher.Matcher {
	return rc.matcher
}

// Cacher interface implementation
func (rc *ResponseCache) Cacher() cache.Cache {
	return rc.cache
}
