package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

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
	Log       LogConfig
	SDK       SDKConfig
	Grpc      GrpcConfig
	Tls       TlsConfig
	Metrics   MetricsConfig
	Http      HttpConfig
	HttpProxy HttpProxyConfig `yaml:"http_proxy"`
}

type SDKConfig struct {
	Key            string `yaml:"key"`
	BaseUrl        string `yaml:"base_url"`
	PollInterval   int    `yaml:"poll_interval"`
	DataGovernance string `yaml:"data_governance"`
	Offline        OfflineConfig
	Cache          CacheConfig
	Log            LogConfig
}

type GrpcConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
	Log     LogConfig
}

type SseConfig struct {
	Enabled           bool              `yaml:"enabled"`
	Headers           map[string]string `yaml:"headers"`
	AllowCORS         bool              `yaml:"allow_cors"`
	HeartBeatInterval int               `yaml:"heart_beat_interval"`
	Log               LogConfig
}

type CertConfig struct {
	Key  string `yaml:"key"`
	Cert string `yaml:"cert"`
}

type HttpConfig struct {
	Port     int            `yaml:"port"`
	CdnProxy CdnProxyConfig `yaml:"cdn_proxy"`
	Log      LogConfig
	Webhook  WebhookConfig
	Sse      SseConfig
	Api      ApiConfig
}

type WebhookConfig struct {
	SignatureValidFor int               `yaml:"signature_valid_for"`
	SigningKey        string            `yaml:"signing_key"`
	AuthHeaders       map[string]string `yaml:"auth_headers"`
	Enabled           bool              `yaml:"enabled"`
	Auth              AuthConfig
}

type CdnProxyConfig struct {
	Headers   map[string]string `yaml:"headers"`
	Enabled   bool              `yaml:"enabled"`
	AllowCORS bool              `yaml:"allow_cors"`
}

type ApiConfig struct {
	AuthHeaders map[string]string `yaml:"auth_headers"`
	Headers     map[string]string `yaml:"headers"`
	Enabled     bool              `yaml:"enabled"`
	AllowCORS   bool              `yaml:"allow_cors"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type HttpProxyConfig struct {
	Url      string `yaml:"url"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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

type CacheConfig struct {
	Redis RedisConfig
}

type RedisConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Addresses []string `yaml:"addresses"`
	DB        int      `yaml:"db"`
	Password  string   `yaml:"password"`
	Tls       TlsConfig
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

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
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
	}

	config.loadEnv()
	if config.Log.GetLevel() == log.None {
		config.Log.Level = "warn"
	}
	config.fixupLogLevels(config.Log.Level)
	return config, nil
}

func (l *LogConfig) GetLevel() log.Level {
	if lvl, ok := allowedLogLevels[l.Level]; ok {
		return lvl
	}
	return log.None
}

func (t *TlsConfig) GetVersion() uint16 {
	if ver, ok := allowedTlsVersions[t.MinVersion]; ok {
		return ver
	}
	return tls.VersionTLS12
}

func (c *Config) setDefaults() {
	c.Http.Port = 8050

	c.Grpc.Enabled = true
	c.Grpc.Port = 50051

	c.Metrics.Enabled = true
	c.Metrics.Port = 8051

	c.Http.Sse.Enabled = true
	c.Http.Sse.AllowCORS = true

	c.Http.CdnProxy.Enabled = true
	c.Http.CdnProxy.AllowCORS = true

	c.Http.Api.Enabled = true
	c.Http.Api.AllowCORS = true

	c.Http.Webhook.Enabled = true
	c.Http.Webhook.SignatureValidFor = 300

	c.SDK.PollInterval = 30
	c.SDK.Offline.Local.PollInterval = 5
	c.SDK.Offline.CachePollInterval = 5

	c.SDK.Cache.Redis.DB = 0
	c.SDK.Cache.Redis.Addresses = []string{"localhost:6379"}
}

func (c *Config) fixupLogLevels(defLevel string) {
	if c.SDK.Log.GetLevel() == log.None {
		c.SDK.Log.Level = defLevel
	}
	if c.SDK.Offline.Log.GetLevel() == log.None {
		c.SDK.Offline.Log.Level = defLevel
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
}
