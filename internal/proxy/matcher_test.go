package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatcherNoCacheParams(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods["test"] = cacheParams{
		cacheByParams:     false,
		paramsInCacheID:   nil,
		paramsInCacheName: nil,
	}
	request := rpcRequest{
		remoteAddr: "",
		JSONRPC:    "",
		ID:         1,
		Method:     "test",
		Params:     []string{"1", "2", "3"},
	}
	key := matcherImp.key(request)
	require.Equal(t, "test", key)
}

func TestMatcherCacheParamsByID(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheID:   []int{0, 2},
		paramsInCacheName: nil,
	}
	var params interface{} = []interface{}{"1", "2", "3"}
	request := rpcRequest{
		remoteAddr: "",
		JSONRPC:    "",
		ID:         1,
		Method:     "test",
		Params:     params,
	}
	key := matcherImp.key(request)
	parts := strings.Split(key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByName(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	}
	var params interface{} = map[string]interface{}{"a": "b", "b": "a"}
	request := rpcRequest{
		remoteAddr: "",
		JSONRPC:    "",
		ID:         1,
		Method:     "test",
		Params:     params,
	}
	key := matcherImp.key(request)
	parts := strings.Split(key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByNameParamsAsJsonList(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	}
	var params interface{} = []interface{}{"1", "2"}
	request := rpcRequest{
		remoteAddr: "",
		JSONRPC:    "",
		ID:         1,
		Method:     "test",
		Params:     params,
	}
	key := matcherImp.key(request)
	require.Equal(t, "", key)
}
