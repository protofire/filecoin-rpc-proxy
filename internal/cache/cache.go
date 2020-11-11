package cache

import (
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/patrickmn/go-cache"
)

type Error struct {
	message string
}

func (e Error) Error() string {
	return e.message
}

type Cache interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
}

type MemoryCache struct {
	*cache.Cache
}

func (m *MemoryCache) Set(key string, value []byte) error {
	m.Cache.Set(key, value, 0)
	return nil
}

func (m *MemoryCache) Get(key string) ([]byte, error) {
	value, ok := m.Cache.Get(key)
	if ok {
		return value.([]byte), nil
	}
	return nil, nil
}

func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		cache.New(defaultExpiration, cleanupInterval),
	}
}

func NewMemoryCacheDefault() *MemoryCache {
	return NewMemoryCache(
		time.Duration(config.DefaultCacheExpiration)*time.Second,
		time.Duration(config.DefaultCacheCleanupInterval)*time.Second,
	)
}

func NewMemoryCacheFromConfig(config *config.Config) *MemoryCache {
	return &MemoryCache{
		cache.New(
			time.Duration(config.CacheSettings.DefaultExpiration)*time.Second,
			time.Duration(config.CacheSettings.CleanupInterval)*time.Second,
		),
	}
}
