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

type cacheMethods []cacheMethod
type methods map[string]cacheMethods

type cacheKey struct {
	Key         string
	cardinality int
}

func (c cacheKey) IsEmpty() bool {
	return c.Key == ""
}

type cacheKeys []cacheKey

func (k cacheKeys) sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].cardinality > k[j].cardinality
	})
}

type customMethod struct {
	Name   string
	Params interface{}
}
type customMethods []customMethod

func (m methods) Custom() customMethods {
	var res customMethods
	for _, cMethods := range m {
		for _, method := range cMethods {
			if method.kind.IsCustom() {
				res = append(res, customMethod{
					Name:   method.name,
					Params: method.paramsForRequest,
				})
			}
		}
	}
	return res
}

type Matcher interface {
	Keys(method string, params interface{}) cacheKeys
	Methods() customMethods
	IsUpdatable(method string) bool
	IsCacheable(method string) bool
}

type cacheMethod struct {
	name              string
	kind              config.MethodType
	cacheByParams     bool
	noStoreCache      bool
	noUpdateCache     bool
	paramsInCacheID   []int
	paramsInCacheName []string
	paramsForRequest  interface{}
}

func (c cacheMethod) match(params interface{}) ([]interface{}, error) {
	if !c.cacheByParams {
		return nil, nil
	}
	var paramsForCache []interface{}
	if len(c.paramsInCacheID) == 0 && len(c.paramsInCacheName) == 0 {
		// cache by all Params
		paramsForCache = append(paramsForCache, params)
		return paramsForCache, nil
	}
	if len(c.paramsInCacheID) > 0 {
		sliceParams, ok := params.([]interface{})
		if ok {
			for idx := range c.paramsInCacheID {
				if idx >= len(sliceParams) {
					return nil, fmt.Errorf("invalid index %d in slice params: %v", idx, sliceParams)
				}
				paramsForCache = append(paramsForCache, sliceParams[idx])
			}
			return paramsForCache, nil
		}
	}
	if len(c.paramsInCacheName) > 0 {
		mapParams, ok := params.(map[string]interface{})
		if ok {
			for _, key := range c.paramsInCacheName {
				param, ok := mapParams[key]
				if !ok {
					return nil, fmt.Errorf("invalid parameter %s key in map: %v", key, mapParams)
				}
				paramsForCache = append(paramsForCache, param)
			}
			return paramsForCache, nil
		}
	}
	return nil, fmt.Errorf("cannot match parameters: %v. matcher: %v", params, c)
}

func (c cacheMethod) toKey(method string, params interface{}) cacheKey {
	key, err := c.match(params)
	if err != nil {
		logger.Log.Error(err)
		return cacheKey{}
	}
	strKey := interfaceSliceToString(key)
	keyParams := []string{method}
	if strKey != "" {
		keyParams = append(keyParams, strKey)
	}
	return cacheKey{Key: strings.Join(keyParams, "_"), cardinality: len(key)}
}

type match struct {
	methods methods
}

func newMatcher() *match {
	userMethods := make(methods)
	return &match{methods: userMethods}
}

func (m *match) IsUpdatable(method string) bool {
	methods, ok := m.methods[method]
	if !ok {
		return false
	}
	for _, m := range methods {
		if m.noUpdateCache {
			return false
		}
	}
	return true
}

func (m *match) IsCacheable(method string) bool {
	methods, ok := m.methods[method]
	if !ok {
		return false
	}
	for _, m := range methods {
		if m.noStoreCache {
			return false
		}
	}
	return true
}

func (m match) addMethod(method config.CacheMethod) {
	if !method.Enabled {
		return
	}
	paramsInCacheName := method.ParamsInCacheByName
	sort.Strings(paramsInCacheName)
	m.methods[method.Name] = append(m.methods[method.Name], cacheMethod{
		kind:              *method.Kind,
		name:              method.Name,
		cacheByParams:     method.CacheByParams,
		paramsInCacheID:   method.ParamsInCacheByID,
		paramsInCacheName: paramsInCacheName,
		noStoreCache:      method.NoStoreCache,
		noUpdateCache:     method.NoUpdateCache,
		paramsForRequest:  method.ParamsForRequest,
	})
}

// FromConfig init match from config
// nolint
func FromConfig(c *config.Config) *match {
	matcher := newMatcher()
	for _, method := range c.CacheMethods {
		matcher.addMethod(method)
	}
	return matcher
}

func (m match) Keys(method string, params interface{}) cacheKeys {
	cacheMethods, ok := m.methods[method]
	if !ok {
		return nil
	}
	var keys cacheKeys
	for _, cm := range cacheMethods {
		if key := cm.toKey(method, params); !key.IsEmpty() {
			keys = append(keys, key)
		}
	}
	if len(keys) > 0 {
		keys.sort()
	}
	return keys
}

func (m match) Methods() customMethods {
	return m.methods.Custom()
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
