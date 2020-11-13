package matcher

import (
	"os"
	"strings"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/stretchr/testify/require"
)

const testMethod = "test"

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
	params := []interface{}{"1", "2", "3"}
	key := matcherImp.Key(testMethod, params)
	require.Equal(t, "test", key)
}

func TestMatcherCacheParamsByID(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheID:   []int{0, 2},
		paramsInCacheName: nil,
	}
	var params interface{} = []interface{}{"1", "2", "3"}
	key := matcherImp.Key(testMethod, params)
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
	var params interface{} = map[string]interface{}{"a": "b", "b": "a"}
	key := matcherImp.Key(testMethod, params)
	parts := strings.Split(key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByNameParamsAsJSONList(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.cacheMethods["test"] = cacheParams{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	}
	var params interface{} = []interface{}{"1", "2"}
	key := matcherImp.Key(testMethod, params)
	require.Equal(t, "", key)
}
