package config

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/model"
	"os"
	"strconv"
	"strings"
)

type envVarReadError struct {
	key   string
	inner error
}

func (e envVarReadError) Error() string {
	return fmt.Sprintf("failed to read environment variable '%s': %s", e.key, e.inner)
}

var envPrefix = "CONFIGCAT"

var toInt = func(s string) (int, error) { return strconv.Atoi(s) }
var toBool = func(s string) (bool, error) { return strconv.ParseBool(s) }
var toFloat = func(s string) (float64, error) { return strconv.ParseFloat(s, 64) }
var toStringSlice = func(s string) ([]string, error) {
	var r []string
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return r, nil
}
var toCertConfigSlice = func(s string) ([]CertConfig, error) {
	var r []CertConfig
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return r, nil
}
var toStringMap = func(s string) (map[string]string, error) {
	var r map[string]string
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return r, nil
}

var toUserAttrs = func(s string) (model.UserAttrs, error) {
	var r model.UserAttrs
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Config) loadEnv() error {
	var sdks map[string]string
	if err := readEnv(envPrefix, "SDKS", &sdks, toStringMap); err != nil {
		return err
	}
	if c.SDKs == nil {
		c.SDKs = make(map[string]*SDKConfig, len(sdks))
	}
	for sdkId, key := range sdks {
		prefix := concatPrefix(envPrefix, strings.ToUpper(strings.ReplaceAll(sdkId, "-", "_")))
		sdkConf := &SDKConfig{Key: key}
		if err := sdkConf.loadEnv(prefix); err != nil {
			return err
		}
		c.SDKs[sdkId] = sdkConf
	}
	if err := c.AutoSDK.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Http.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Grpc.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.HttpProxy.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Log.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Tls.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Diag.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.Cache.loadEnv(envPrefix); err != nil {
		return err
	}
	if err := c.GlobalOfflineConfig.loadEnv(envPrefix); err != nil {
		return err
	}

	return readEnv(envPrefix, "DEFAULT_USER_ATTRIBUTES", &c.DefaultAttrs, toUserAttrs)
}

func (s *SDKConfig) loadEnv(prefix string) error {
	readEnvString(prefix, "BASE_URL", &s.BaseUrl)
	readEnvString(prefix, "DATA_GOVERNANCE", &s.DataGovernance)
	readEnvString(prefix, "WEBHOOK_SIGNING_KEY", &s.WebhookSigningKey)
	if err := readEnv(prefix, "WEBHOOK_SIGNATURE_VALID_FOR", &s.WebhookSignatureValidFor, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "POLL_INTERVAL", &s.PollInterval, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "DEFAULT_USER_ATTRIBUTES", &s.DefaultAttrs, toUserAttrs); err != nil {
		return err
	}
	if err := s.Offline.loadEnv(prefix); err != nil {
		return err
	}
	return s.Log.loadEnv(prefix)
}

func (a *AutoSDKConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "AUTO_CONFIG")
	readEnvString(prefix, "KEY", &a.Key)
	readEnvString(prefix, "SECRET", &a.Secret)
	readEnvString(prefix, "BASE_URL", &a.BaseUrl)
	readEnvString(prefix, "SDK_BASE_URL", &a.SdkBaseUrl)
	readEnvString(prefix, "WEBHOOK_SIGNING_KEY", &a.WebhookSigningKey)
	if err := readEnv(prefix, "WEBHOOK_SIGNATURE_VALID_FOR", &a.WebhookSignatureValidFor, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "POLL_INTERVAL", &a.PollInterval, toInt); err != nil {
		return err
	}
	return a.Log.loadEnv(prefix)
}

func (h *HttpConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "HTTP")
	if err := readEnv(prefix, "ENABLED", &h.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "PORT", &h.Port, toInt); err != nil {
		return err
	}
	if err := h.Log.loadEnv(prefix); err != nil {
		return err
	}
	if err := h.Sse.loadEnv(prefix); err != nil {
		return err
	}
	if err := h.CdnProxy.loadEnv(prefix); err != nil {
		return err
	}
	if err := h.Webhook.loadEnv(prefix); err != nil {
		return err
	}
	if err := h.Status.loadEnv(prefix); err != nil {
		return err
	}
	if err := h.OFREP.loadEnv(prefix); err != nil {
		return err
	}
	return h.Api.loadEnv(prefix)
}

func (g *GrpcConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "GRPC")
	if err := readEnv(prefix, "ENABLED", &g.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "PORT", &g.Port, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEALTH_CHECK_ENABLED", &g.HealthCheckEnabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "SERVER_REFLECTION_ENABLED", &g.ServerReflectionEnabled, toBool); err != nil {
		return err
	}
	if err := g.KeepAlive.loadEnv(prefix); err != nil {
		return err
	}
	return g.Log.loadEnv(prefix)
}

func (k *KeepAliveConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "KEEP_ALIVE")
	if err := readEnv(prefix, "MAX_CONNECTION_IDLE", &k.MaxConnectionIdle, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "MAX_CONNECTION_AGE", &k.MaxConnectionAge, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "MAX_CONNECTION_AGE_GRACE", &k.MaxConnectionAgeGrace, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "TIME", &k.Time, toInt); err != nil {
		return err
	}
	return readEnv(prefix, "TIMEOUT", &k.Timeout, toInt)
}

func (h *HttpProxyConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "HTTP_PROXY")
	readEnvString(prefix, "URL", &h.Url)
	return nil
}

func (c *CacheConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "CACHE")
	if err := c.Redis.loadEnv(prefix); err != nil {
		return err
	}
	if err := c.MongoDb.loadEnv(prefix); err != nil {
		return err
	}
	return c.DynamoDb.loadEnv(prefix)
}

func (g *GlobalOfflineConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "OFFLINE")
	if err := readEnv(prefix, "ENABLED", &g.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "CACHE_POLL_INTERVAL", &g.CachePollInterval, toInt); err != nil {
		return err
	}
	return g.Log.loadEnv(prefix)
}

func (o *OfflineConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "OFFLINE")
	if err := readEnv(prefix, "ENABLED", &o.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "USE_CACHE", &o.UseCache, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "CACHE_POLL_INTERVAL", &o.CachePollInterval, toInt); err != nil {
		return err
	}
	if err := o.Local.loadEnv(prefix); err != nil {
		return err
	}
	return o.Log.loadEnv(prefix)
}

func (l *LocalConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "LOCAL")
	if err := readEnv(prefix, "POLLING", &l.Polling, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "POLL_INTERVAL", &l.PollInterval, toInt); err != nil {
		return err
	}
	readEnvString(prefix, "FILE_PATH", &l.FilePath)
	return nil
}

func (r *RedisConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "REDIS")
	readEnvString(prefix, "PASSWORD", &r.Password)
	readEnvString(prefix, "USER", &r.User)
	if err := readEnv(prefix, "DB", &r.DB, toInt); err != nil {
		return err
	}
	if err := readEnv(prefix, "ENABLED", &r.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "ADDRESSES", &r.Addresses, toStringSlice); err != nil {
		return err
	}
	return r.Tls.loadEnv(prefix)
}

func (m *MongoDbConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "MONGODB")
	readEnvString(prefix, "URL", &m.Url)
	readEnvString(prefix, "DATABASE", &m.Database)
	readEnvString(prefix, "COLLECTION", &m.Collection)
	if err := readEnv(prefix, "ENABLED", &m.Enabled, toBool); err != nil {
		return err
	}
	return m.Tls.loadEnv(prefix)
}

func (d *DynamoDbConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "DYNAMODB")
	readEnvString(prefix, "URL", &d.Url)
	readEnvString(prefix, "TABLE", &d.Table)
	return readEnv(prefix, "ENABLED", &d.Enabled, toBool)
}

func (s *SseConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "SSE")
	if err := readEnv(prefix, "ENABLED", &s.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEADERS", &s.Headers, toStringMap); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEARTBEAT_INTERVAL", &s.HeartBeatInterval, toInt); err != nil {
		return err
	}
	if err := s.CORS.loadEnv(prefix); err != nil {
		return err
	}
	return s.Log.loadEnv(prefix)
}

func (a *ApiConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "API")
	if err := readEnv(prefix, "ENABLED", &a.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEADERS", &a.Headers, toStringMap); err != nil {
		return err
	}
	if err := readEnv(prefix, "AUTH_HEADERS", &a.AuthHeaders, toStringMap); err != nil {
		return err
	}
	return a.CORS.loadEnv(prefix)
}

func (o *OFREPConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "OFREP")
	if err := readEnv(prefix, "ENABLED", &o.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEADERS", &o.Headers, toStringMap); err != nil {
		return err
	}
	if err := readEnv(prefix, "AUTH_HEADERS", &o.AuthHeaders, toStringMap); err != nil {
		return err
	}
	return o.CORS.loadEnv(prefix)
}

func (c *CdnProxyConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "CDN_PROXY")
	if err := readEnv(prefix, "ENABLED", &c.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "HEADERS", &c.Headers, toStringMap); err != nil {
		return err
	}
	return c.CORS.loadEnv(prefix)
}

func (w *WebhookConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "WEBHOOK")
	if err := readEnv(prefix, "ENABLED", &w.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "AUTH_HEADERS", &w.AuthHeaders, toStringMap); err != nil {
		return err
	}
	return w.Auth.loadEnv(prefix)
}

func (a *AuthConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "AUTH")
	readEnvString(prefix, "USER", &a.User)
	readEnvString(prefix, "PASSWORD", &a.Password)
	return nil
}

func (c *CORSConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "CORS")
	if err := readEnv(prefix, "ENABLED", &c.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "ALLOWED_ORIGINS", &c.AllowedOrigins, toStringSlice); err != nil {
		return err
	}
	return c.AllowedOriginsRegex.loadEnv(prefix)
}

func (o *OriginRegexConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "ALLOWED_ORIGINS_REGEX")
	readEnvString(prefix, "IF_NO_MATCH", &o.IfNoMatch)
	return readEnv(prefix, "PATTERNS", &o.Patterns, toStringSlice)
}

func (t *TlsConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "TLS")
	readEnvString(prefix, "SERVER_NAME", &t.ServerName)
	if err := readEnv(prefix, "MIN_VERSION", &t.MinVersion, toFloat); err != nil {
		return err
	}
	if err := readEnv(prefix, "ENABLED", &t.Enabled, toBool); err != nil {
		return err
	}
	return readEnv(prefix, "CERTIFICATES", &t.Certificates, toCertConfigSlice)
}

func (l *LogConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "LOG")
	readEnvString(prefix, "LEVEL", &l.Level)
	return nil
}

func (d *DiagConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "DIAG")
	if err := readEnv(prefix, "ENABLED", &d.Enabled, toBool); err != nil {
		return err
	}
	if err := readEnv(prefix, "PORT", &d.Port, toInt); err != nil {
		return err
	}
	if err := d.Status.loadEnv(prefix); err != nil {
		return err
	}
	if err := d.Metrics.loadEnv(prefix); err != nil {
		return err
	}
	return nil
}

func (m *MetricsConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "METRICS")
	if err := readEnv(prefix, "ENABLED", &m.Enabled, toBool); err != nil {
		return err
	}
	return nil
}

func (s *StatusConfig) loadEnv(prefix string) error {
	prefix = concatPrefix(prefix, "STATUS")
	if err := readEnv(prefix, "ENABLED", &s.Enabled, toBool); err != nil {
		return err
	}
	return nil
}

func readEnv[T any](prefix string, key string, in *T, conv func(string) (T, error)) error {
	envKey := prefix + "_" + key
	if env := os.Getenv(envKey); env != "" {
		r, err := conv(env)
		if err != nil {
			return &envVarReadError{key: envKey, inner: err}
		}
		*in = r
	}
	return nil
}

func readEnvString(prefix string, key string, in *string) {
	if env := os.Getenv(prefix + "_" + key); env != "" {
		*in = env
	}
}

func concatPrefix(p1 string, p2 string) string {
	return p1 + "_" + p2
}
