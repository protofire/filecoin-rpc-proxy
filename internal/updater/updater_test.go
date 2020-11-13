package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

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

func TestMain(t *testing.M) {
	logger.InitDefaultLogger()
	os.Exit(t.Run())
}

func TestUpdater(t *testing.T) {

	method := "test"
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

	conf, err := testhelpers.GetConfig(backend.URL, method)
	require.NoError(t, err)

	var params interface{} = []interface{}{"1", "2"}
	conf.CacheMethods[0].ParamsForRequest = params

	cacher := proxy.NewResponseCache(
		cache.NewMemoryCacheFromConfig(conf),
		matcher.FromConfig(conf),
	)
	updaterImp, err := FromConfig(conf, cacher, logger.Log)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go updaterImp.Start(ctx, 1)
	cancel()

	ctxStop, cancel := context.WithTimeout(context.Background(), time.Second*1)
	updaterImp.StopWithTimeout(ctxStop)
	defer cancel()

	lock.Lock()
	require.GreaterOrEqual(t, requestsCount, 1)
	lock.Unlock()

	reqs := updaterImp.requests()
	cachedResp, err := updaterImp.cacher.GetResponseCache(reqs[0])
	require.NoError(t, err)
	require.False(t, cachedResp.IsEmpty())
	require.True(t, utils.Equal(cachedResp.ID, response.ID))

}
