package matcher

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

type method struct {
	Name   string
	Params interface{}
}

type Matcher interface {
	Key(method string, params interface{}) string
	Methods() []method
}

type cacheParams struct {
	cacheByParams     bool
	paramsInCacheID   []int
	paramsInCacheName []string
	paramsForRequest  interface{}
}

func (p cacheParams) match(params interface{}) ([]interface{}, error) {
	if !p.cacheByParams {
		return nil, nil
	}
	var paramsForCache []interface{}
	if len(p.paramsInCacheID) == 0 && len(p.paramsInCacheName) == 0 {
		// cache by all Params
		paramsForCache = append(paramsForCache, params)
		return paramsForCache, nil
	}
	if len(p.paramsInCacheID) != 0 {
		sliceParams, ok := params.([]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse testMethod parameters %#v with cache Params by ID: %#v", params, p.paramsInCacheID)
		}
		for idx := range p.paramsInCacheID {
			paramsForCache = append(paramsForCache, sliceParams[idx])
		}
		return paramsForCache, nil
	}
	mapParams, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse testMethod parameters %#v with cache Params by Name: %#v", params, p.paramsInCacheName)
	}
	for _, key := range p.paramsInCacheName {
		paramsForCache = append(paramsForCache, mapParams[key])
	}
	return paramsForCache, nil
}

type match struct {
	cacheMethods map[string]cacheParams
	methods      []method
}

func newMatcher() *match {
	methods := make(map[string]cacheParams)
	return &match{cacheMethods: methods}
}

func (m match) addCacheMethod(method config.CacheMethod) {
	paramsInCacheName := method.ParamsInCacheByName
	sort.Strings(paramsInCacheName)
	m.cacheMethods[method.Name] = cacheParams{
		cacheByParams:     method.CacheByParams,
		paramsInCacheID:   method.ParamsInCacheByID,
		paramsInCacheName: paramsInCacheName,
		paramsForRequest:  method.ParamsForRequest,
	}
}

// FromConfig init match from config
// nolint
func FromConfig(c *config.Config) *match {
	matcher := newMatcher()
	for _, method := range c.CacheMethods {
		matcher.addCacheMethod(method)
	}
	for name, methods := range matcher.cacheMethods {
		matcher.methods = append(matcher.methods, method{
			Name:   name,
			Params: methods.paramsForRequest,
		})
	}
	return matcher
}

func (m match) Key(method string, params interface{}) string {
	cacheParams, ok := m.cacheMethods[method]
	if !ok {
		return ""
	}
	key, err := cacheParams.match(params)
	if err != nil {
		logger.Log.Error(err)
		return ""
	}
	strKey := interfaceSliceToString(key)
	keyParams := []string{method}
	if strKey != "" {
		keyParams = append(keyParams, strKey)
	}
	return strings.Join(keyParams, "_")
}

func (m match) Methods() []method {
	return m.methods
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
