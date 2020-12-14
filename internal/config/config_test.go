package config

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	proxyURL         = "http://test.com"
	token            = "token"
	methodName       = "test"
	paramInCacheID   = 1
	redisURI         = "redis://127.0.0.1:6379"
	paramInCacheName = "field"
	configParamsByID = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_methods:
- name: %s
  kind: custom	
  cache_by_params: true
  params_for_request:
    - one
    - three
    - two
  params_in_cache_by_id:
    - %s
`, proxyURL, token, methodName, strconv.Itoa(paramInCacheID))
	configParamsByName = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_methods:
- name: %s
  kind: custom	
  cache_by_params: true
  params_for_request:
    - 1
    - one
    - two
  params_in_cache_by_name:
    - %s
`, proxyURL, token, methodName, paramInCacheName)
	configParamsByIDAndName = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_methods:
- name: %s
  cache_by_params: true
  params_for_request:
    - 1
    - one
    - two
  params_in_cache_by_id:
    - %s
  params_in_cache_by_name:
    - %s
`, proxyURL, token, methodName, strconv.Itoa(paramInCacheID), paramInCacheName)
	configParamsByIDAndNameRegular = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_settings:
  storage: %s
  redis:
    uri: %s
cache_methods:
- name: %s
  cache_by_params: true
  params_in_cache_by_id:
    - %s
  params_in_cache_by_name:
    - %s
`, proxyURL, token, RedisCacheStorage, redisURI, methodName, strconv.Itoa(paramInCacheID), paramInCacheName)
	configParamsByIDAndNameWrongMethodKind = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_methods:
- name: %s
  kind: kind
  cache_by_params: true
  params_in_cache_by_id:
    - %s
  params_in_cache_by_name:
    - %s
`, proxyURL, token, methodName, strconv.Itoa(paramInCacheID), paramInCacheName)
	configParamsWrongCacheStorage = fmt.Sprintf(`
proxy_url: %s
jwt_secret: %s
cache_settings:
	storage: %s	
cache_methods:
- name: %s
  cache_by_params: true
  params_in_cache_by_id:
    - %s
  params_in_cache_by_name:
    - %s
`, proxyURL, token, "xxx", methodName, strconv.Itoa(paramInCacheID), paramInCacheName)
)

func TestNewConfigCacheParamsByID(t *testing.T) {
	config, err := New(strings.NewReader(configParamsByID))
	require.NoError(t, err, err)
	require.Equal(t, config.ProxyURL, proxyURL)
	require.True(t, config.CacheMethods[0].CacheByParams)
	require.Equal(t, config.CacheMethods[0].Name, methodName)
	require.Equal(t, config.CacheMethods[0].ParamsInCacheByID[0], paramInCacheID)
	require.Equal(t, config.CacheSettings.Memory.DefaultExpiration, 0)
	require.Equal(t, config.CacheSettings.Memory.CleanupInterval, -1)
	require.True(t, config.CacheMethods[0].Kind.IsCustom())
}

func TestNewConfigCacheParamsByName(t *testing.T) {
	config, err := New(strings.NewReader(configParamsByName))
	require.NoError(t, err, err)
	require.Equal(t, config.ProxyURL, proxyURL)
	require.True(t, config.CacheMethods[0].CacheByParams)
	require.Equal(t, config.CacheMethods[0].Name, methodName)
	require.Equal(t, config.CacheMethods[0].ParamsInCacheByName[0], paramInCacheName)
	require.True(t, config.CacheMethods[0].Kind.IsCustom())
}

func TestNewConfigCacheParamsByIDAndName(t *testing.T) {
	config, err := New(strings.NewReader(configParamsByIDAndName))
	require.NoError(t, err, err)
	require.True(t, config.CacheMethods[0].Kind.IsCustom())
	require.True(t, config.CacheSettings.Storage.IsMemory())
}

func TestNewConfigCacheParamsByIDAndNameRegular(t *testing.T) {
	config, err := New(strings.NewReader(configParamsByIDAndNameRegular))
	require.NoError(t, err, err)
	require.True(t, config.CacheMethods[0].Kind.IsRegular())
	require.True(t, config.CacheSettings.Storage.IsRedis())
}

func TestNewConfigCacheParamsWrongCacheStorage(t *testing.T) {
	_, err := New(strings.NewReader(configParamsWrongCacheStorage))
	require.Error(t, err, err)
}

func TestNewConfigCacheParamsByIDWrongMethodKind(t *testing.T) {
	_, err := New(strings.NewReader(configParamsByIDAndNameWrongMethodKind))
	require.Error(t, err, err)
}
