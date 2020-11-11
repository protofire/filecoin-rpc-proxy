package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	// in seconds
	DefaultCacheCleanupInterval = -1
	DefaultCacheExpiration      = 0
	defaultLogLevel             = "INFO"
	defaultPort                 = 8080
	defaultHost                 = "0.0.0.0"
)

type CacheMethod struct {
	Name              string   `yaml:"name"`
	CacheByParams     bool     `yaml:"cache_by_params,omitempty"`
	ParamsInCacheID   []int    `yaml:"params_in_cache_id,omitempty"`
	ParamsInCacheName []string `yaml:"params_in_cache_name,omitempty"`
}

type CacheSettings struct {
	DefaultExpiration int `yaml:"expiration,omitempty"`
	CleanupInterval   int `yaml:"cleanup_interval,omitempty"`
}

type Config struct {
	CacheMethods   []CacheMethod `yaml:"cache_methods,omitempty"`
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
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
	return c, c.validate()
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
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.Host == "" {
		c.Host = defaultHost
	}
}

func (c *Config) validate() error {
	for _, params := range c.CacheMethods {
		if len(params.ParamsInCacheID) > 0 && len(params.ParamsInCacheName) > 0 {
			return fmt.Errorf("either cache params by ID or cache params by name are supported")
		}
	}
	return nil
}

func NewConfigFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return NewConfig(file)
}
