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
	paramInCacheName = "field"
	configParamsByID = fmt.Sprintf(`
proxy_url: %s
jwt_token: %s
jwt_secret: %s
cache_methods:
- name: %s
  cache_by_params: true
  params_for_request:
    - one
    - three
    - two
  params_in_cache_by_id:
    - %s
`, proxyURL, token, token, methodName, strconv.Itoa(paramInCacheID))
	configParamsByName = fmt.Sprintf(`
proxy_url: %s
jwt_token: %s
jwt_secret: %s
cache_methods:
- name: %s
  cache_by_params: true
  params_for_request:
    - 1
    - one
    - two
  params_in_cache_by_name:
    - %s
`, proxyURL, token, token, methodName, paramInCacheName)
	configParamsByIDAndName = fmt.Sprintf(`
proxy_url: %s
jwt_token: %s
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
`, proxyURL, token, token, methodName, strconv.Itoa(paramInCacheID), paramInCacheName)
)

func TestNewConfigCacheParamsByID(t *testing.T) {
	config, err := NewConfig(strings.NewReader(configParamsByID))
	require.NoError(t, err)
	require.Equal(t, config.ProxyURL, proxyURL)
	require.True(t, config.CacheMethods[0].CacheByParams)
	require.Equal(t, config.CacheMethods[0].Name, methodName)
	require.Equal(t, config.CacheMethods[0].ParamsInCacheByID[0], paramInCacheID)
	require.Equal(t, config.CacheSettings.DefaultExpiration, 0)
	require.Equal(t, config.CacheSettings.CleanupInterval, -1)
}

func TestNewConfigCacheParamsByName(t *testing.T) {
	config, err := NewConfig(strings.NewReader(configParamsByName))
	require.NoError(t, err)
	require.Equal(t, config.ProxyURL, proxyURL)
	require.True(t, config.CacheMethods[0].CacheByParams)
	require.Equal(t, config.CacheMethods[0].Name, methodName)
	require.Equal(t, config.CacheMethods[0].ParamsInCacheByName[0], paramInCacheName)
}

func TestNewConfigCacheParamsByIDAndName(t *testing.T) {
	_, err := NewConfig(strings.NewReader(configParamsByIDAndName))
	require.Error(t, err)
}
