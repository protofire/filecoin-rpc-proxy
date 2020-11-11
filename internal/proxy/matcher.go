package proxy

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

type matcher interface {
	key(request rpcRequest) string
}

type cacheParams struct {
	cacheByParams     bool
	paramsInCacheID   []int
	paramsInCacheName []string
}

func (p cacheParams) match(params interface{}) ([]interface{}, error) {
	if !p.cacheByParams {
		return nil, nil
	}
	var paramsForCache []interface{}
	if len(p.paramsInCacheID) == 0 && len(p.paramsInCacheName) == 0 {
		// cache by all params
		paramsForCache = append(paramsForCache, params)
		return paramsForCache, nil
	}
	if len(p.paramsInCacheID) != 0 {
		sliceParams, ok := params.([]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse method parameters %#v with cache params by ID: %#v", params, p.paramsInCacheID)
		}
		for idx := range p.paramsInCacheID {
			paramsForCache = append(paramsForCache, sliceParams[idx])
		}
		return paramsForCache, nil
	}
	mapParams, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse method parameters %#v with cache params by Name: %#v", params, p.paramsInCacheName)
	}
	for _, key := range p.paramsInCacheName {
		paramsForCache = append(paramsForCache, mapParams[key])
	}
	return paramsForCache, nil
}

type match struct {
	methods map[string]cacheParams
}

func newMatcher() *match {
	methods := make(map[string]cacheParams)
	return &match{methods: methods}
}

// NewMatcherFromConfig init match from config
func NewMatcherFromConfig(c *config.Config) *match {
	matcher := newMatcher()
	for _, method := range c.CacheMethods {
		paramsInCacheName := method.ParamsInCacheName
		sort.Strings(paramsInCacheName)
		matcher.methods[method.Name] = cacheParams{
			cacheByParams:     method.CacheByParams,
			paramsInCacheID:   method.ParamsInCacheID,
			paramsInCacheName: paramsInCacheName,
		}
	}
	return matcher
}

func (m match) key(r rpcRequest) string {
	cacheParams, ok := m.methods[r.Method]
	if !ok {
		return ""
	}
	key, err := cacheParams.match(r.Params)
	if err != nil {
		logger.Log.Error(err)
		return ""
	}
	strKey := interfaceSliceToString(key)
	params := []string{r.Method}
	if strKey != "" {
		params = append(params, strKey)
	}
	return strings.Join(params, "_")
}

func interfaceSliceToString(params []interface{}) string {
	if len(params) == 0 {
		return ""
	}
	hash := sha256.New()
	for _, ifs := range params {
		value, _ := json.Marshal(ifs)
		_, _ = hash.Write(value)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
