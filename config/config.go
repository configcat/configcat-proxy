package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

const defaultConfigName = "options.yml"
const defaultVendorName = "configcat"
const defaultProductName = "proxy"

var allowedLogLevels = map[string]log.Level{
	"debug": log.Debug,
	"info":  log.Info,
	"warn":  log.Warn,
	"error": log.Error,
}

var allowedTlsVersions = map[float64]uint16{
	1.0: tls.VersionTLS10,
	1.1: tls.VersionTLS11,
	1.2: tls.VersionTLS12,
	1.3: tls.VersionTLS13,
}

type Config struct {
	Log                 LogConfig
	SDKs                map[string]*SDKConfig
	Grpc                GrpcConfig
	Tls                 TlsConfig
	Diag                DiagConfig
	Http                HttpConfig
	Cache               CacheConfig
	HttpProxy           HttpProxyConfig     `yaml:"http_proxy"`
	GlobalOfflineConfig GlobalOfflineConfig `yaml:"offline"`
	DefaultAttrs        model.UserAttrs     `yaml:"default_user_attributes"`
}

type SDKConfig struct {
	Key                      string          `yaml:"key"`
	BaseUrl                  string          `yaml:"base_url"`
	PollInterval             int             `yaml:"poll_interval"`
	DataGovernance           string          `yaml:"data_governance"`
	WebhookSignatureValidFor int             `yaml:"webhook_signature_valid_for"`
	WebhookSigningKey        string          `yaml:"webhook_signing_key"`
	DefaultAttrs             model.UserAttrs `yaml:"default_user_attributes"`
	Offline                  OfflineConfig
	Log                      LogConfig
}

type GrpcConfig struct {
	Enabled                 bool            `yaml:"enabled"`
	Port                    int             `yaml:"port"`
	ServerReflectionEnabled bool            `yaml:"server_reflection_enabled"`
	HealthCheckEnabled      bool            `yaml:"health_check_enabled"`
	KeepAlive               KeepAliveConfig `yaml:"keep_alive"`
	Log                     LogConfig
}

type KeepAliveConfig struct {
	MaxConnectionIdle     int `yaml:"max_connection_idle"`
	MaxConnectionAge      int `yaml:"max_connection_age"`
	MaxConnectionAgeGrace int `yaml:"max_connection_age_grace"`
	Time                  int `yaml:"time"`
	Timeout               int `yaml:"timeout"`
}

type SseConfig struct {
	Enabled           bool              `yaml:"enabled"`
	Headers           map[string]string `yaml:"headers"`
	HeartBeatInterval int               `yaml:"heart_beat_interval"`
	Log               LogConfig
	CORS              CORSConfig
}

type CertConfig struct {
	Key  string `yaml:"key"`
	Cert string `yaml:"cert"`
}

type HttpConfig struct {
	Enabled  bool
	Port     int            `yaml:"port"`
	CdnProxy CdnProxyConfig `yaml:"cdn_proxy"`
	Log      LogConfig
	Webhook  WebhookConfig
	Sse      SseConfig
	Api      ApiConfig
	Status   StatusConfig
}

type WebhookConfig struct {
	AuthHeaders map[string]string `yaml:"auth_headers"`
	Enabled     bool              `yaml:"enabled"`
	Auth        AuthConfig
}

type CdnProxyConfig struct {
	Headers map[string]string `yaml:"headers"`
	Enabled bool              `yaml:"enabled"`
	CORS    CORSConfig
}

type ApiConfig struct {
	AuthHeaders map[string]string `yaml:"auth_headers"`
	Headers     map[string]string `yaml:"headers"`
	Enabled     bool              `yaml:"enabled"`
	CORS        CORSConfig
}

type CORSConfig struct {
	Enabled             bool
	AllowedOrigins      []string          `yaml:"allowed_origins"`
	AllowedOriginsRegex OriginRegexConfig `yaml:"allowed_origins_regex"`
}

type OriginRegexConfig struct {
	IfNoMatch string `yaml:"if_no_match"`
	Patterns  []string

	Regexes []*regexp.Regexp `yaml:"-"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type HttpProxyConfig struct {
	Url string `yaml:"url"`
}

type AuthConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type OfflineConfig struct {
	Enabled           bool `yaml:"enabled"`
	UseCache          bool `yaml:"use_cache"`
	CachePollInterval int  `yaml:"cache_poll_interval"`
	Log               LogConfig
	Local             LocalConfig
}

type GlobalOfflineConfig struct {
	Enabled           bool `yaml:"enabled"`
	CachePollInterval int  `yaml:"cache_poll_interval"`
	Log               LogConfig
}

type CacheConfig struct {
	Redis    RedisConfig
	MongoDb  MongoDbConfig  `yaml:"mongodb"`
	DynamoDb DynamoDbConfig `yaml:"dynamodb"`
}

type RedisConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Addresses []string `yaml:"addresses"`
	DB        int      `yaml:"db"`
	User      string   `yaml:"user"`
	Password  string   `yaml:"password"`
	Tls       TlsConfig
}

type MongoDbConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Url        string `yaml:"url"`
	Database   string `yaml:"database"`
	Collection string `yaml:"collection"`
	Tls        TlsConfig
}

type DynamoDbConfig struct {
	Enabled bool   `yaml:"enabled"`
	Url     string `yaml:"url"`
	Table   string `yaml:"table"`
}

type LocalConfig struct {
	FilePath     string `yaml:"file_path"`
	Polling      bool   `yaml:"polling"`
	PollInterval int    `yaml:"poll_interval"`
}

type TlsConfig struct {
	Enabled      bool    `yaml:"enabled"`
	MinVersion   float64 `yaml:"min_version"`
	ServerName   string  `yaml:"server_name"`
	Certificates []CertConfig
}

type DiagConfig struct {
	Port    int           `yaml:"port"`
	Enabled bool          `yaml:"enabled"`
	Metrics MetricsConfig `yaml:"metrics"`
	Status  StatusConfig  `yaml:"status"`
}

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type StatusConfig struct {
	Enabled bool `yaml:"enabled"`
}

func LoadConfigFromFileAndEnvironment(filePath string) (Config, error) {
	var config Config
	config.setDefaults()

	if filePath != "" {
		_, err := os.Stat(filePath)
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("config file %s does not exist: %s", filePath, err)
		}
		realPath, err := filepath.EvalSymlinks(filePath)
		if err != nil {
			return Config{}, fmt.Errorf("failed to eval symlink for %s: %s", realPath, err)
		}
		data, err := os.ReadFile(realPath)
		if err != nil {
			return Config{}, fmt.Errorf("failed to read config file %s: %s", realPath, err)
		}
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return Config{}, fmt.Errorf("failed to parse YAML from config file %s: %s", realPath, err)
		}
	} else if defaultPath, ok := defaultConfigPath(); ok {
		_, err := os.Stat(defaultPath)
		if !errors.Is(err, os.ErrNotExist) {
			data, err := os.ReadFile(defaultPath)
			if err != nil {
				return Config{}, fmt.Errorf("failed to read config file %s: %s", defaultPath, err)
			}
			err = yaml.Unmarshal(data, &config)
			if err != nil {
				return Config{}, fmt.Errorf("failed to parse YAML from config file %s: %s", defaultPath, err)
			}
		}
	}

	if err := config.loadEnv(); err != nil {
		return Config{}, err
	}
	if config.Log.GetLevel() == log.None {
		config.Log.Level = "warn"
	}
	config.fixupLogLevels(config.Log.Level)
	config.fixupDefaults()
	config.fixupTlsMinVersions(1.2)
	config.fixupOffline()
	if err := config.compileOriginRegexes(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (l *LogConfig) GetLevel() log.Level {
	if lvl, ok := allowedLogLevels[l.Level]; ok {
		return lvl
	}
	return log.None
}

func (t *TlsConfig) GetVersion() uint16 {
	return allowedTlsVersions[t.MinVersion]
}

func (c *Config) setDefaults() {
	c.Http.Port = 8050
	c.Http.Enabled = true

	c.Grpc.Enabled = true
	c.Grpc.Port = 50051
	c.Grpc.HealthCheckEnabled = true
	c.Grpc.ServerReflectionEnabled = false

	c.Diag.Enabled = true
	c.Diag.Status.Enabled = true
	c.Diag.Metrics.Enabled = true
	c.Diag.Port = 8051

	c.Http.Sse.Enabled = true
	c.Http.Sse.CORS.Enabled = true

	c.Http.CdnProxy.Enabled = true
	c.Http.CdnProxy.CORS.Enabled = true

	c.Http.Api.Enabled = true
	c.Http.Api.CORS.Enabled = true

	c.Http.Webhook.Enabled = true

	c.Http.Status.Enabled = false

	c.Cache.Redis.DB = 0
	c.Cache.Redis.Addresses = []string{"localhost:6379"}

	c.Cache.MongoDb.Database = "configcat_proxy"
	c.Cache.MongoDb.Collection = "cache"

	c.Cache.DynamoDb.Table = "configcat_proxy_cache"
}

func (c *Config) fixupDefaults() {
	for _, sdk := range c.SDKs {
		if sdk == nil {
			continue
		}
		if sdk.WebhookSignatureValidFor == 0 {
			sdk.WebhookSignatureValidFor = 300
		}
		if sdk.PollInterval == 0 {
			sdk.PollInterval = 30
		}
		if sdk.Offline.Local.PollInterval == 0 {
			sdk.Offline.Local.PollInterval = 5
		}
		if sdk.Offline.CachePollInterval == 0 {
			sdk.Offline.CachePollInterval = 5
		}
	}
	if c.GlobalOfflineConfig.CachePollInterval == 0 {
		c.GlobalOfflineConfig.CachePollInterval = 5
	}
}

func (c *Config) fixupOffline() {
	if !c.GlobalOfflineConfig.Enabled {
		return
	}
	for _, sdk := range c.SDKs {
		if sdk == nil {
			continue
		}
		if !sdk.Offline.Enabled {
			sdk.Offline.Enabled = true
			sdk.Offline.UseCache = true
			sdk.Offline.CachePollInterval = c.GlobalOfflineConfig.CachePollInterval
			sdk.Offline.Log = c.GlobalOfflineConfig.Log
		}
	}
}

func (c *Config) fixupLogLevels(defLevel string) {
	for _, sdk := range c.SDKs {
		if sdk == nil {
			continue
		}
		if sdk.Log.GetLevel() == log.None {
			sdk.Log.Level = defLevel
		}
		if sdk.Offline.Log.GetLevel() == log.None {
			sdk.Offline.Log.Level = defLevel
		}
	}
	if c.Http.Log.GetLevel() == log.None {
		c.Http.Log.Level = defLevel
	}
	if c.Http.Sse.Log.GetLevel() == log.None {
		c.Http.Sse.Log.Level = defLevel
	}
	if c.Grpc.Log.GetLevel() == log.None {
		c.Grpc.Log.Level = defLevel
	}
	if c.GlobalOfflineConfig.Log.GetLevel() == log.None {
		c.GlobalOfflineConfig.Log.Level = defLevel
	}
}

func (c *Config) fixupTlsMinVersions(defVersion float64) {
	if _, ok := allowedTlsVersions[c.Tls.MinVersion]; !ok {
		c.Tls.MinVersion = defVersion
	}
	if _, ok := allowedTlsVersions[c.Cache.Redis.Tls.MinVersion]; !ok {
		c.Cache.Redis.Tls.MinVersion = defVersion
	}
	if _, ok := allowedTlsVersions[c.Cache.MongoDb.Tls.MinVersion]; !ok {
		c.Cache.MongoDb.Tls.MinVersion = defVersion
	}
}

func (c *Config) compileOriginRegexes() error {
	if err := c.Http.Api.CORS.compileRegexes(); err != nil {
		return err
	}
	if err := c.Http.CdnProxy.CORS.compileRegexes(); err != nil {
		return err
	}
	if err := c.Http.Sse.CORS.compileRegexes(); err != nil {
		return err
	}
	return nil
}

func (c *CORSConfig) compileRegexes() error {
	if !c.Enabled {
		return nil
	}
	if c.AllowedOriginsRegex.Patterns != nil && len(c.AllowedOriginsRegex.Patterns) > 0 {
		for _, pattern := range c.AllowedOriginsRegex.Patterns {
			reg, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			c.AllowedOriginsRegex.Regexes = append(c.AllowedOriginsRegex.Regexes, reg)
		}
	}
	return nil
}

func (k *KeepAliveConfig) ToParams() (keepalive.ServerParameters, bool) {
	if k.MaxConnectionIdle == 0 && k.MaxConnectionAge == 0 && k.MaxConnectionAgeGrace == 0 &&
		k.Time == 0 && k.Timeout == 0 {
		return keepalive.ServerParameters{}, false
	}
	param := keepalive.ServerParameters{}
	if k.MaxConnectionIdle != 0 {
		param.MaxConnectionIdle = time.Duration(k.MaxConnectionIdle) * time.Second
	}
	if k.MaxConnectionAge != 0 {
		param.MaxConnectionAge = time.Duration(k.MaxConnectionAge) * time.Second
	}
	if k.MaxConnectionAgeGrace != 0 {
		param.MaxConnectionAgeGrace = time.Duration(k.MaxConnectionAgeGrace) * time.Second
	}
	if k.Time != 0 {
		param.Time = time.Duration(k.Time) * time.Second
	}
	if k.Timeout != 0 {
		param.Timeout = time.Duration(k.Timeout) * time.Second
	}
	return param, true
}

func (c *CacheConfig) IsSet() bool {
	return c.Redis.Enabled || c.MongoDb.Enabled || c.DynamoDb.Enabled
}

func (t *TlsConfig) LoadTlsOptions() (*tls.Config, error) {
	conf := &tls.Config{
		MinVersion: t.GetVersion(),
		ServerName: t.ServerName,
	}
	for _, c := range t.Certificates {
		if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
			conf.Certificates = append(conf.Certificates, cert)
		} else {
			return nil, fmt.Errorf("failed to load the certificate and key files: %s", err)
		}
	}
	return conf, nil
}

func defaultConfigPath() (string, bool) {
	switch runtime.GOOS {
	case "windows":
		rootDir := os.Getenv("PROGRAMDATA")
		return path.Join(rootDir, defaultVendorName, defaultProductName, defaultConfigName), true
	case "darwin":
		return path.Join("/Library/Application Support", defaultVendorName, defaultProductName, defaultConfigName), true
	case "linux":
		return path.Join("/etc", defaultVendorName, defaultProductName, defaultConfigName), true
	}
	return "", false
}
