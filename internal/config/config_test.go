package config

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	proxyURL     = "http://test.com"
	methodName   = "test"
	paramInCache = 1
	config       = fmt.Sprintf(`
proxy_url: %s
cache_methods:
- name: %s
  cache_by_params: true
  params_in_cache:
    - %s
`, proxyURL, methodName, strconv.Itoa(paramInCache))
)

func TestNewConfig(t *testing.T) {
	config, err := NewConfig(strings.NewReader(config))
	require.NoError(t, err)
	require.Equal(t, config.ProxyURL, proxyURL)
	require.True(t, config.CacheMethods[0].CacheByParams)
	require.Equal(t, config.CacheMethods[0].Name, methodName)
	require.Equal(t, config.CacheMethods[0].ParamsInCache[0], paramInCache)
	require.Equal(t, config.CacheSettings.DefaultExpiration, 0)
	require.Equal(t, config.CacheSettings.CleanupInterval, -1)
}
