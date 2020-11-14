package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type MethodType string

const (
	// in seconds
	DefaultCacheCleanupInterval            = -1
	DefaultCacheExpiration                 = 0
	defaultLogLevel                        = "INFO"
	defaultPort                            = 8080
	defaultHost                            = "0.0.0.0"
	defaultJWTAlgorithm                    = "HS256"
	defaultSystemCachePeriod               = 600
	defaultUserCachePeriod                 = 3600
	defaultRequestsBatchSize               = 5
	defaultRequestsConcurrency             = 10
	CustomMethod                MethodType = "custom"
	RegularMethod               MethodType = "regular"
)

func (t MethodType) IsCustom() bool {
	return t == CustomMethod
}

func (t MethodType) IsRegular() bool {
	return t == RegularMethod
}

func (t MethodType) Valid() error {
	switch t {
	case CustomMethod, RegularMethod:
		return nil
	default:
		return fmt.Errorf("unknown method type: %s", t)
	}
}

func (t *MethodType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	if err := unmarshal(&buf); err != nil {
		return err
	}
	newT := MethodType(buf)
	if err := newT.Valid(); err != nil {
		return err
	}
	*t = newT
	return nil
}

func (t MethodType) MarshalYAML() (interface{}, error) {
	return string(t), nil
}

type CacheMethod struct {
	Name                string      `yaml:"name"`
	CacheByParams       bool        `yaml:"cache_by_params,omitempty"`
	ParamsInCacheByID   []int       `yaml:"params_in_cache_by_id,omitempty"`
	ParamsInCacheByName []string    `yaml:"params_in_cache_by_name,omitempty"`
	Kind                *MethodType `yaml:"kind,omitempty"`
	ParamsForRequest    interface{} `yaml:"params_for_request,omitempty"`
}

type CacheSettings struct {
	DefaultExpiration int `yaml:"expiration,omitempty"`
	CleanupInterval   int `yaml:"cleanup_interval,omitempty"`
}

type Config struct {
	CacheMethods            []CacheMethod `yaml:"cache_methods,omitempty"`
	JWTAlgorithm            string        `yaml:"jwt_alg"`
	JWTSecret               string        `yaml:"jwt_secret"`
	Host                    string        `yaml:"host"`
	Port                    int           `yaml:"port"`
	UpdateSystemCachePeriod int           `yaml:"update_system_cache_period"`
	UpdateUserCachePeriod   int           `yaml:"update_user_cache_period"`
	RequestsBatchSize       int           `yaml:"requests_batch_size"`
	RequestsConcurrency     int           `yaml:"requests_concurrency"`
	ProxyURL                string        `yaml:"proxy_url"`
	CacheSettings           CacheSettings `yaml:"cache_settings,omitempty"`
	LogLevel                string        `yaml:"log_level"`
	LogPrettyPrint          bool          `yaml:"log_pretty_print"`
}

func New(reader io.Reader) (*Config, error) {
	c := &Config{}
	if err := yaml.NewDecoder(reader).Decode(c); err != nil {
		return nil, err
	}
	c.Init()
	return c, c.Validate()
}

func (c *Config) Init() {
	if c.CacheSettings.CleanupInterval == 0 {
		c.CacheSettings.CleanupInterval = DefaultCacheCleanupInterval
	}
	if c.CacheSettings.DefaultExpiration == 0 {
		c.CacheSettings.DefaultExpiration = DefaultCacheExpiration
	}
	if c.LogLevel == "" {
		c.LogLevel = defaultLogLevel
	}
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.Host == "" {
		c.Host = defaultHost
	}
	if c.JWTAlgorithm == "" {
		c.JWTAlgorithm = defaultJWTAlgorithm
	}
	if c.UpdateSystemCachePeriod == 0 {
		c.UpdateSystemCachePeriod = defaultSystemCachePeriod
	}
	if c.UpdateUserCachePeriod == 0 {
		c.UpdateUserCachePeriod = defaultUserCachePeriod
	}
	if c.RequestsBatchSize == 0 {
		c.RequestsBatchSize = defaultRequestsBatchSize
	}
	if c.RequestsConcurrency == 0 {
		c.RequestsConcurrency = defaultRequestsConcurrency
	}
	for idx := range c.CacheMethods {
		method := c.CacheMethods[idx]
		if method.Kind == nil {
			if method.ParamsForRequest == nil {
				mt := RegularMethod
				method.Kind = &mt
			} else {
				mt := CustomMethod
				method.Kind = &mt
			}
			c.CacheMethods[idx] = method
		}
	}
}

func (c *Config) Validate() error {
	for _, method := range c.CacheMethods {
		if err := method.Kind.Valid(); err != nil {
			return err
		}
		if method.Kind.IsCustom() && method.ParamsForRequest == nil {
			return fmt.Errorf("custom method type should have been set ParamsForRequest")
		}
		if method.Kind.IsRegular() && method.ParamsForRequest != nil {
			return fmt.Errorf("regular method type should not have been set ParamsForRequest")
		}
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt secret is mandatory parameter")
	}
	return nil
}

func FromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return New(file)
}
