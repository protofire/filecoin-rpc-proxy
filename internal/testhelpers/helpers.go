package testhelpers

import (
	"os"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

var (
	token    = "token"
	logLevel = "INFO"
)

func init() {
	logLevel = os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}
}

func GetConfig(url string, methods ...string) (*config.Config, error) {
	conf := config.Config{
		JWTSecret: token,
		ProxyURL:  url,
		LogLevel:  logLevel,
	}

	for _, method := range methods {
		conf.CacheMethods = append(conf.CacheMethods, config.CacheMethod{
			Name:          method,
			CacheByParams: true,
			Enabled:       true,
		})
	}
	conf.Init()

	return &conf, conf.Validate()

}

func GetRedisConfig(url string, redisURI string, methods ...string) (*config.Config, error) {
	conf := config.Config{
		JWTSecret: token,
		ProxyURL:  url,
		LogLevel:  logLevel,
		CacheSettings: config.CacheSettings{
			Storage: config.RedisCacheStorage,
			Redis:   config.RedisCacheSettings{URI: redisURI},
		},
	}

	for _, method := range methods {
		conf.CacheMethods = append(conf.CacheMethods, config.CacheMethod{
			Name:          method,
			CacheByParams: true,
			Enabled:       true,
		})
	}
	conf.Init()

	return &conf, conf.Validate()

}

func GetConfigWithCustomMethods(url string, methods ...string) (*config.Config, error) {
	conf := config.Config{
		JWTSecret: token,
		ProxyURL:  url,
		LogLevel:  logLevel,
	}

	for _, method := range methods {
		mt := config.CustomMethod
		conf.CacheMethods = append(conf.CacheMethods, config.CacheMethod{
			Name:             method,
			CacheByParams:    true,
			Kind:             &mt,
			ParamsForRequest: []interface{}{},
			Enabled:          true,
		})
	}
	conf.Init()

	return &conf, conf.Validate()

}
