package proxy

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

type matcher interface {
	key(request rpcRequest) string
}

type params struct {
	cacheByParams bool
	paramsInCache []int
}

type match struct {
	methods map[string]params
}

func newMatcher() *match {
	methods := make(map[string]params)
	return &match{methods: methods}
}

// NewMatcherFromConfig init match from config
func NewMatcherFromConfig(c *config.Config) *match {
	matcher := newMatcher()
	for _, method := range c.CacheMethods {
		matcher.methods[method.Name] = params{
			cacheByParams: method.CacheByParams,
			paramsInCache: method.ParamsInCache,
		}
	}
	return matcher
}

func (m match) key(r rpcRequest) string {
	params, ok := m.methods[r.Method]
	if !ok {
		return ""
	}
	if !params.cacheByParams {
		return r.Method
	}
	var paramsForCache []json.RawMessage
	if len(params.paramsInCache) == 0 {
		paramsForCache = r.Params
	} else {
		for idx := range params.paramsInCache {
			paramsForCache = append(paramsForCache, r.Params[idx])
		}
	}
	return rawMessagesToString(paramsForCache)
}

func rawMessageToString(message json.RawMessage) string {
	hash := md5.New()
	hash.Write(bytes.ReplaceAll(message, []byte(" "), []byte("")))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func rawMessagesToString(messages []json.RawMessage) string {
	res := make([]string, len(messages))
	for idx, message := range messages {
		res[idx] = rawMessageToString(message)
	}
	return strings.Join(res, "_")
}
