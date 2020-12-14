package matcher

import (
	"os"
	"strings"
	"testing"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"go.uber.org/goleak"

	"github.com/stretchr/testify/require"
)

const testMethod = "test"

func TestMain(t *testing.M) { // nolint
	logger.InitDefaultLogger()
	goleak.VerifyTestMain(t)
	os.Exit(t.Run())
}

func TestMatcherNoCacheParams(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     false,
		paramsInCacheID:   nil,
		paramsInCacheName: nil,
	})
	params := []interface{}{"1", "2", "3"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 1)
	require.Equal(t, "test", keys[0].Key)
}

func TestMatcherCacheParamsByID(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     true,
		paramsInCacheID:   []int{0, 2},
		paramsInCacheName: nil,
	})
	var params interface{} = []interface{}{"1", "2", "3"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 1)
	parts := strings.Split(keys[0].Key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByName(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   nil,
	})
	var params interface{} = map[string]interface{}{"a": "b", "b": "a"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 1)
	parts := strings.Split(keys[0].Key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestMatcherCacheParamsByNameParamsAsJSONList(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
	})
	var params interface{} = []interface{}{"1", "2"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 0)
}

func TestMatcherCacheParamsByNameWrongParams(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
	})
	var params interface{} = map[string]interface{}{"c": "b", "d": "a"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 0)
}

func TestMatcherCacheParamsByNameAndByID(t *testing.T) {
	matcherImp := newMatcher()
	matcherImp.methods[testMethod] = append(matcherImp.methods[testMethod], cacheMethod{
		cacheByParams:     true,
		paramsInCacheName: []string{"a", "b"},
		paramsInCacheID:   []int{1, 2},
	})
	var params interface{} = map[string]interface{}{"a": "b", "b": "a"}
	keys := matcherImp.Keys(testMethod, params)
	require.Len(t, keys, 1)
	parts := strings.Split(keys[0].Key, "_")
	require.Equal(t, "test", parts[0])
	require.Len(t, parts, 2)
}

func TestKeys(t *testing.T) {
	allKeys := cacheKeys{{Key: "1", cardinality: 1}, {Key: "2", cardinality: 2}, {Key: "3", cardinality: 100}}
	allKeys.sort()
	require.Equal(t, "3", allKeys[0].Key)
	require.Equal(t, "2", allKeys[1].Key)
	require.Equal(t, "1", allKeys[2].Key)
}
