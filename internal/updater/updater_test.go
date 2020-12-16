package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"

	"github.com/protofire/filecoin-rpc-proxy/internal/utils"

	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"
	"golang.org/x/net/context"

	"github.com/protofire/filecoin-rpc-proxy/internal/requests"

	"github.com/protofire/filecoin-rpc-proxy/internal/testhelpers"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { // nolint
	logger.InitDefaultLogger()
}

const method = "test"

func TestMethodsUpdater(t *testing.T) {

	requestID := 1
	result := float64(15)

	response := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}
	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)

	requestsCount := 0
	lock := sync.Mutex{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		lock.Lock()
		requestsCount++
		lock.Unlock()
		_, err := fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfigWithCustomMethods(backend.URL, method)
	require.NoError(t, err)

	var params interface{} = []interface{}{"1", "2"}
	conf.CacheMethods[0].ParamsForRequest = params

	cacher := proxy.NewResponseCache(
		cache.NewMemoryCacheFromConfig(conf.CacheSettings.Memory),
		matcher.FromConfig(conf),
	)
	updaterImp, err := FromConfig(conf, cacher, logger.Log)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go updaterImp.StartMethodUpdater(ctx, 1)
	cancel()

	ctxStop, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	updaterImp.StopWithTimeout(ctxStop, 1)
	defer cancel()

	lock.Lock()
	require.GreaterOrEqual(t, requestsCount, 1)
	lock.Unlock()

	reqs := updaterImp.methodRequests()
	require.NotEqual(t, 0, len(reqs))
	cachedResp, err := updaterImp.cacher.GetResponseCache(reqs[0])
	require.NoError(t, err)
	require.False(t, cachedResp.IsEmpty())
	require.True(t, utils.Equal(cachedResp.ID, response.ID))

}

func TestCacheUpdater(t *testing.T) {

	requestID := 1
	result := float64(15)

	var params interface{} = []interface{}{"1", "2"}
	request := requests.RPCRequest{
		Method:  method,
		JSONRPC: "2.0",
		ID:      requestID,
		Params:  params,
	}
	response := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}
	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)

	requestsCount := 0
	lock := sync.Mutex{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		lock.Lock()
		requestsCount++
		lock.Unlock()
		_, err := fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfigWithCustomMethods(backend.URL, method)
	require.NoError(t, err)

	cacher := proxy.NewResponseCache(
		cache.NewMemoryCacheFromConfig(conf.CacheSettings.Memory),
		matcher.FromConfig(conf),
	)
	updaterImp, err := FromConfig(conf, cacher, logger.Log)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	err = updaterImp.cacher.SetResponseCache(request, response)
	require.NoError(t, err)

	go updaterImp.StartCacheUpdater(ctx, 1)
	cancel()

	ctxStop, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	updaterImp.StopWithTimeout(ctxStop, 1)
	defer cancel()

	lock.Lock()
	require.GreaterOrEqual(t, requestsCount, 1)
	lock.Unlock()

	cachedResp, err := updaterImp.cacher.GetResponseCache(request)
	require.NoError(t, err)
	require.False(t, cachedResp.IsEmpty())
	require.True(t, utils.Equal(cachedResp.ID, response.ID))

}

func TestRedisCacheUpdater(t *testing.T) {

	requestID := 1
	result := float64(15)

	var params interface{} = []interface{}{"1", "2"}
	request := requests.RPCRequest{
		Method:  method,
		JSONRPC: "2.0",
		ID:      requestID,
		Params:  params,
	}
	response := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}
	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)

	requestsCount := 0
	lock := sync.Mutex{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		lock.Lock()
		requestsCount++
		lock.Unlock()
		_, err := fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetRedisConfig(backend.URL, testhelpers.RedisURI, method)
	require.NoError(t, err)

	ctx, cancelRedis := context.WithCancel(context.Background())
	cacheImpl, err := cache.FromConfig(ctx, conf)
	require.NoError(t, err)

	cacher := proxy.NewResponseCache(cacheImpl, matcher.FromConfig(conf))
	updaterImp, err := FromConfig(conf, cacher, logger.Log)
	require.NoError(t, err)

	err = updaterImp.cacher.SetResponseCache(request, response)
	require.NoError(t, err)

	go updaterImp.StartCacheUpdater(ctx, 1)
	reqs := updaterImp.cacheRequests()
	require.Len(t, reqs, 1)

	ctxStop, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	updaterImp.StopWithTimeout(ctxStop, 1)

	defer func() {
		cancelRedis()
		cancel()
		if err := cacheImpl.Clean(); err != nil {
			logger.Log.Error(err)
		}
		if err := cacheImpl.Close(); err != nil {
			logger.Log.Error(err)
		}
	}()

	lock.Lock()
	require.GreaterOrEqual(t, requestsCount, 1)
	lock.Unlock()

	cachedResp, err := updaterImp.cacher.GetResponseCache(request)
	require.NoError(t, err)
	require.False(t, cachedResp.IsEmpty())
	require.True(t, utils.Equal(cachedResp.ID, response.ID))

}

func TestMethodsUpdaterConcurrency(t *testing.T) {

	requestID := 1
	result := float64(15)
	n := 100

	response := requests.RPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
		Error:   nil,
	}
	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)

	var requestsCount []int
	lock := sync.Mutex{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		reqs, err := requests.ParseRequests(r)
		require.NoError(t, err)
		lock.Lock()
		requestsCount = append(requestsCount, len(reqs))
		lock.Unlock()
		_, err = fmt.Fprint(w, string(responseJSON))
		if err != nil {
			logger.Log.Error(err)
		}
	}))
	defer backend.Close()

	conf, err := testhelpers.GetConfigWithCustomMethods(backend.URL, method)
	require.NoError(t, err)

	var params interface{} = []interface{}{"1", "2"}
	conf.CacheMethods[0].ParamsForRequest = params
	kind := config.CustomMethod
	for i := 1; i < n; i++ {
		conf.CacheMethods = append(conf.CacheMethods, config.CacheMethod{
			Name:             fmt.Sprintf("method%d", i),
			CacheByParams:    true,
			Kind:             &kind,
			ParamsForRequest: params,
			Enabled:          true,
		})
	}

	require.Len(t, conf.CacheMethods, n)

	cacher := proxy.NewResponseCache(
		cache.NewMemoryCacheFromConfig(conf.CacheSettings.Memory),
		matcher.FromConfig(conf),
	)
	updaterImp, err := FromConfig(conf, cacher, logger.Log)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go updaterImp.StartMethodUpdater(ctx, 1)

	counter := 50

	for {
		lock.Lock()
		l := len(requestsCount)
		lock.Unlock()
		if l == n/updaterImp.batchSize || counter <= 0 {
			break
		}
		counter--
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	ctxStop, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	updaterImp.StopWithTimeout(ctxStop, 1)
	defer cancel()

	lock.Lock()
	require.Equal(t, n/updaterImp.batchSize, len(requestsCount), requestsCount)
	for _, v := range requestsCount {
		require.LessOrEqual(t, v, updaterImp.batchSize)
	}
	lock.Unlock()

}
