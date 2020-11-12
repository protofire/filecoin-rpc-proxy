package cache

import (
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"

	"github.com/patrickmn/go-cache"
)

// Error for cache package
type Error struct {
	message string
}

func (e Error) Error() string {
	return e.message
}

// Cache ...
type Cache interface {
	Set(key string, value interface{}) error
	Get(key string) (interface{}, error)
}

// MemoryCache ...
type MemoryCache struct {
	*cache.Cache
}

// Set ...
func (m *MemoryCache) Set(key string, value interface{}) error {
	m.Cache.Set(key, value, 0)
	metrics.SetCacheSize(int64(m.Cache.ItemCount()))
	return nil
}

// Get ...
func (m *MemoryCache) Get(key string) (interface{}, error) {
	value, ok := m.Cache.Get(key)
	if ok {
		return value, nil
	}
	return nil, nil
}

// NewMemoryCache initializes memory cache
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		cache.New(defaultExpiration, cleanupInterval),
	}
}

// NewMemoryCacheDefault initializes memory cache with default parameters
func NewMemoryCacheDefault() *MemoryCache {
	return NewMemoryCache(
		time.Duration(config.DefaultCacheExpiration)*time.Second,
		time.Duration(config.DefaultCacheCleanupInterval)*time.Second,
	)
}

// NewMemoryCacheFromConfig initializes memory cache from config
func NewMemoryCacheFromConfig(config *config.Config) *MemoryCache {
	return &MemoryCache{
		cache.New(
			time.Duration(config.CacheSettings.DefaultExpiration)*time.Second,
			time.Duration(config.CacheSettings.CleanupInterval)*time.Second,
		),
	}
}
