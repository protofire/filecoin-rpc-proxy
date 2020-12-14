package cache

import (
	"testing"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/stretchr/testify/require"
)

func TestNewMemoryCacheDefault(t *testing.T) {
	cache := NewMemoryCacheDefault()
	expectedRequest := requests.RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "",
		Params:  nil,
	}
	expectedResponse := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  nil,
		Error:   nil,
	}
	err := cache.Set("1", expectedRequest, expectedResponse)
	require.NoError(t, err)
	value, err := cache.Get("1")
	require.NoError(t, err)
	require.Equal(t, expectedResponse, value)
}

func TestNewMemoryCacheExpired(t *testing.T) {
	d := time.Duration(1) * time.Second
	cache := NewMemoryCache(d, -1)
	expectedRequest := requests.RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "",
		Params:  nil,
	}
	expectedResponse := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  nil,
		Error:   nil,
	}
	err := cache.Set("1", expectedRequest, expectedResponse)
	require.NoError(t, err)
	time.Sleep(d)
	value, err := cache.Get("1")
	require.NoError(t, err)
	require.True(t, value.IsEmpty())
}
