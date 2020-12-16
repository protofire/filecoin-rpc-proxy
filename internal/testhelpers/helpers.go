package testhelpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ory/dockertest/v3/docker"
	"github.com/protofire/filecoin-rpc-proxy/internal/cache"

	"github.com/ory/dockertest/v3"
	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"go.uber.org/goleak"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
)

var (
	logLevel = "INFO"
	RedisURI = fmt.Sprintf("redis://%s:%s", redisHost, redisPort)
)

const (
	token     = "token"
	redisHost = "127.0.0.1"
	redisPort = "6379"
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

func TestMain(m *testing.M) { // nolint

	logger.InitDefaultLogger()

	pool, err := dockertest.NewPool("")
	if err != nil {
		logger.Log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   "redis",
		Tag:          "latest",
		ExposedPorts: []string{redisPort},
		PortBindings: map[docker.Port][]docker.PortBinding{
			redisPort: {{HostIP: redisHost, HostPort: redisPort}},
		},
	})

	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	ctx := context.Background()
	if err = pool.Retry(func() error {
		var err error
		c, err := cache.NewRedisClient(ctx, config.RedisCacheSettings{
			URI: RedisURI,
		})
		if err == nil {
			_ = c.Close()
		}
		return err
	}); err != nil {
		log.Fatalf("Could not connect to redis: %s", err)
	}

	exitCode := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	if exitCode == 0 {
		if err := goleak.Find(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: Errors on successful test run: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)

}
