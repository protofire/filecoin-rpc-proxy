package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMemoryCacheDefault(t *testing.T) {
	cache := NewMemoryCacheDefault()
	expectedValue := []byte("cache")
	err := cache.Set("1", expectedValue)
	require.NoError(t, err)
	value, err := cache.Get("1")
	require.NoError(t, err)
	require.Equal(t, expectedValue, value)
}
func TestNewMemoryCacheExpired(t *testing.T) {
	d := time.Duration(1) * time.Second
	cache := NewMemoryCache(d, -1)
	expectedValue := []byte("cache")
	err := cache.Set("1", expectedValue)
	require.NoError(t, err)
	time.Sleep(d)
	value, err := cache.Get("1")
	require.NoError(t, err)
	require.Nil(t, value)
	require.Len(t, value, 0)
}
