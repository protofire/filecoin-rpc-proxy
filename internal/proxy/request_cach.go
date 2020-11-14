package proxy

import (
	"encoding/json"

	"github.com/hashicorp/go-multierror"

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
	Cacher() cache.Cache
}

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

func (rc *ResponseCache) GetResponseCache(req requests.RPCRequest) (requests.RPCResponse, error) {
	resp := requests.RPCResponse{}
	keys := rc.matcher.Keys(req.Method, req.Params)
	if len(keys) == 0 {
		return resp, nil
	}
	mErr := &multierror.Error{}
	for _, key := range keys {
		data, err := rc.cache.Get(key.Key)
		if err != nil {
			mErr = multierror.Append(mErr, err)
			continue
		}
		if data == nil {
			continue
		}
		resp, ok := data.(requests.RPCResponse)
		if ok {
			return resp, nil
		}
		dataBytes, ok := data.([]byte)
		if !ok {
			continue
		}
		err = json.Unmarshal(dataBytes, &resp)
		if err != nil {
			continue
		}
		break
	}
	return resp, nil
}

func (rc *ResponseCache) Matcher() matcher.Matcher {
	return rc.matcher
}

func (rc *ResponseCache) Cacher() cache.Cache {
	return rc.cache
}
