package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

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

type cacheValue struct {
	Request  requests.RPCRequest
	Response requests.RPCResponse
}

// Cache ...
type Cache interface {
	Set(key string, request requests.RPCRequest, response requests.RPCResponse) error
	Get(key string) (requests.RPCResponse, error)
	Requests() ([]requests.RPCRequest, error)
	Close() error
	Clean() error
}

// MemoryCache ...
type MemoryCache struct {
	*cache.Cache
}

func (m *MemoryCache) Requests() ([]requests.RPCRequest, error) {
	res := make([]requests.RPCRequest, m.Cache.ItemCount())
	for _, item := range m.Cache.Items() {
		res = append(res, item.Object.(cacheValue).Request)
	}
	return res, nil
}

// Set ...
func (m *MemoryCache) Set(key string, request requests.RPCRequest, response requests.RPCResponse) error {
	m.Cache.Set(key, cacheValue{
		Request:  request,
		Response: response,
	}, 0)
	metrics.SetCacheSize(int64(m.Cache.ItemCount()))
	return nil
}

// Get ...
func (m *MemoryCache) Get(key string) (requests.RPCResponse, error) {
	val, ok := m.Cache.Get(key)
	if ok {
		return val.(cacheValue).Response, nil
	}
	return requests.RPCResponse{}, nil
}

// Close ...
func (m *MemoryCache) Close() error {
	m.Cache = nil
	return nil
}

// Clean ...
func (m *MemoryCache) Clean() error {
	m.Cache = nil
	return nil
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
func NewMemoryCacheFromConfig(config config.MemoryCacheSettings) *MemoryCache {
	return &MemoryCache{
		cache.New(
			time.Duration(config.DefaultExpiration)*time.Second,
			time.Duration(config.CleanupInterval)*time.Second,
		),
	}
}

// FromConfig initializes cache from config
func FromConfig(ctx context.Context, c *config.Config) (Cache, error) {
	switch c.CacheSettings.Storage {
	case config.MemoryCacheStorage:
		return NewMemoryCacheFromConfig(c.CacheSettings.Memory), nil
	case config.RedisCacheStorage:
		client, err := NewRedisClient(ctx, c.CacheSettings.Redis)
		if err != nil {
			return nil, err
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unknown cache storage type: %s", c.CacheSettings.Storage)
	}
}
