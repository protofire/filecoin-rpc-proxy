package matcher

import (
	"os"
	"strings"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.M) {
	logger.InitDefaultLogger()
	os.Exit(t.Run())
}

func TestMatcherNoCacheParams(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     false,
		paramsInCacheID:   nil,
		paramsInCacheName: nil,
	}
	method := "test"
	params := []interface{}{"1", "2", "3"}
	key := matcherImp.Key(method, params)
	require.Equal(t, "test", key)
}

func TestMatcherCacheParamsByID(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheID:   []int{0, 2},
		paramsInCacheName: nil,
	}
	method := "test"
	var params interface{} = []interface{}{"1", "2", "3"}
	key := matcherImp.Key(method, params)
	parts := strings.Split(key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByName(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	}
	method := "test"
	var params interface{} = map[string]interface{}{"a": "b", "b": "a"}
	key := matcherImp.Key(method, params)
	parts := strings.Split(key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByNameParamsAsJsonList(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	}
	method := "test"
	var params interface{} = []interface{}{"1", "2"}
	key := matcherImp.Key(method, params)
	require.Equal(t, "", key)
}
