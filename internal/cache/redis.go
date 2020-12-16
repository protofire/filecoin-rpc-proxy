package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"gopkg.in/mgo.v2/bson"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/go-redis/redis/v8"
)

const hashMapName = "filecoin"

// Client represents redis client
type Client struct {
	*redis.Client
}

// NewRedisClient creates redis client
func NewRedisClient(ctx context.Context, config config.RedisCacheSettings) (*Client, error) {
	var opts *redis.Options
	var err error
	var tlsConfig *tls.Config
	if strings.HasSuffix(config.URI, "rediss") {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint
		}
	}
	opts, err = redis.ParseURL(config.URI)
	if err != nil {
		return nil, err
	}
	opts.PoolSize = config.PoolSize
	opts.TLSConfig = tlsConfig

	client := redis.NewClient(opts).WithContext(ctx)
	if _, err = client.Ping(client.Context()).Result(); err != nil {
		return nil, fmt.Errorf("cannot initialize redis client: %w", err)
	}
	return &Client{
		Client: client,
	}, nil
}

func (client *Client) Get(key string) (requests.RPCResponse, error) {
	val := cacheValue{}
	data, err := client.Client.HGet(client.Context(), hashMapName, key).Bytes()
	if err != nil {
		return val.Response, err
	}
	if err := bson.Unmarshal(data, &val); err != nil {
		return val.Response, err
	}
	return val.Response, nil
}

func (client *Client) Set(key string, request requests.RPCRequest, response requests.RPCResponse) error {
	item := cacheValue{
		Request:  request,
		Response: response,
	}
	data, err := bson.Marshal(item)
	if err != nil {
		return err
	}
	return client.Client.HSet(client.Context(), hashMapName, key, data).Err()
}

func (client *Client) Requests() ([]requests.RPCRequest, error) {
	data, err := client.Client.HVals(client.Context(), hashMapName).Result()
	if err != nil {
		return nil, err
	}
	res := make([]requests.RPCRequest, len(data))
	for idx, value := range data {
		item := cacheValue{}
		if err := bson.Unmarshal([]byte(value), &item); err != nil {
			return nil, err
		}
		res[idx] = item.Request
	}
	return res, nil
}

// Close closes redis client
func (client *Client) Close() error {
	if err := client.Client.Close(); err != nil {
		return fmt.Errorf("cannot close redis client %w", err)
	}
	return nil
}

// Clean cleans all cache
func (client *Client) Clean() error {
	if err := client.Client.FlushAll(client.Context()).Err(); err != nil {
		return fmt.Errorf("cannot flush redis database %w", err)
	}
	return nil
}
