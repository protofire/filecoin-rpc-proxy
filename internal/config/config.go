package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	// in seconds
	DefaultCacheCleanupInterval = -1
	DefaultCacheExpiration      = 0
	defaultLogLevel             = "INFO"
)

type CacheMethod struct {
	Name          string `yaml:"name"`
	CacheByParams bool   `yaml:"cache_by_params,omitempty"`
	ParamsInCache []int  `yaml:"params_in_cache,omitempty"`
}

type CacheSettings struct {
	DefaultExpiration int `yaml:"expiration,omitempty"`
	CleanupInterval   int `yaml:"cleanup_interval,omitempty"`
}

type Config struct {
	CacheMethods   []CacheMethod `yaml:"cache_methods,omitempty"`
	ProxyURL       string        `yaml:"proxy_url"`
	CacheSettings  CacheSettings `yaml:"cache_settings,omitempty"`
	LogLevel       string        `yaml:"log_level"`
	LogPrettyPrint bool          `yaml:"log_pretty_print"`
}

func NewConfig(reader io.Reader) (*Config, error) {
	c := &Config{}
	if err := yaml.NewDecoder(reader).Decode(c); err != nil {
		return nil, err
	}
	c.init()
	return c, nil
}

func (c *Config) init() {
	if c.CacheSettings.CleanupInterval == 0 {
		c.CacheSettings.CleanupInterval = DefaultCacheCleanupInterval
	}
	if c.CacheSettings.DefaultExpiration == 0 {
		c.CacheSettings.DefaultExpiration = DefaultCacheExpiration
	}
	if c.LogLevel == "" {
		c.LogLevel = defaultLogLevel
	}
}

func NewConfigFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return NewConfig(file)
}
