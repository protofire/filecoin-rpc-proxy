package testhelpers

import (
	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

var (
	token = "token"
)

func GetConfig(url string, methods ...string) (*config.Config, error) {
	conf := config.Config{
		JWTSecret: token,
		ProxyURL:  url,
	}

	for _, method := range methods {
		conf.CacheMethods = append(conf.CacheMethods, config.CacheMethod{
			Name:          method,
			CacheByParams: true,
		})
	}
	conf.Init()

	return &conf, conf.Validate()

}
