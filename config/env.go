package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

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

func (c *Config) loadEnv() {
	var sdks map[string]string
	readEnv(envPrefix, "SDKS", &sdks, toStringMap)
	if c.SDKs == nil {
		c.SDKs = make(map[string]*SDKConfig, len(sdks))
	}
	for sdkId, key := range sdks {
		prefix := concatPrefix(envPrefix, strings.ToUpper(strings.ReplaceAll(sdkId, "-", "_")))
		sdkConf := &SDKConfig{Key: key}
		sdkConf.loadEnv(prefix)
		c.SDKs[sdkId] = sdkConf
	}
	c.Http.loadEnv(envPrefix)
	c.Grpc.loadEnv(envPrefix)
	c.HttpProxy.loadEnv(envPrefix)
	c.Log.loadEnv(envPrefix)
	c.Tls.loadEnv(envPrefix)
	c.Metrics.loadEnv(envPrefix)
	c.Cache.loadEnv(envPrefix)
	c.GlobalOfflineConfig.loadEnv(envPrefix)

	readEnv(envPrefix, "DEFAULT_USER_ATTRIBUTES", &c.DefaultAttrs, toStringMap)
}

func (s *SDKConfig) loadEnv(prefix string) {
	readEnvString(prefix, "BASE_URL", &s.BaseUrl)
	readEnvString(prefix, "DATA_GOVERNANCE", &s.DataGovernance)
	readEnvString(prefix, "WEBHOOK_SIGNING_KEY", &s.WebhookSigningKey)
	readEnv(prefix, "WEBHOOK_SIGNATURE_VALID_FOR", &s.WebhookSignatureValidFor, toInt)
	readEnv(prefix, "POLL_INTERVAL", &s.PollInterval, toInt)
	readEnv(prefix, "DEFAULT_USER_ATTRIBUTES", &s.DefaultAttrs, toStringMap)
	s.Offline.loadEnv(prefix)
	s.Log.loadEnv(prefix)
}

func (h *HttpConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "HTTP")
	readEnv(prefix, "PORT", &h.Port, toInt)
	h.Log.loadEnv(prefix)
	h.Sse.loadEnv(prefix)
	h.CdnProxy.loadEnv(prefix)
	h.Webhook.loadEnv(prefix)
	h.Api.loadEnv(prefix)
}

func (g *GrpcConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "GRPC")
	readEnv(prefix, "ENABLED", &g.Enabled, toBool)
	readEnv(prefix, "PORT", &g.Port, toInt)
	g.Log.loadEnv(prefix)
}

func (h *HttpProxyConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "HTTP_PROXY")
	readEnvString(prefix, "URL", &h.Url)
}

func (c *CacheConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "CACHE")
	c.Redis.loadEnv(prefix)
}

func (g *GlobalOfflineConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "OFFLINE")
	readEnv(prefix, "ENABLED", &g.Enabled, toBool)
	readEnv(prefix, "CACHE_POLL_INTERVAL", &g.CachePollInterval, toInt)
	g.Log.loadEnv(prefix)
}

func (o *OfflineConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "OFFLINE")
	readEnv(prefix, "ENABLED", &o.Enabled, toBool)
	readEnv(prefix, "USE_CACHE", &o.UseCache, toBool)
	readEnv(prefix, "CACHE_POLL_INTERVAL", &o.CachePollInterval, toInt)
	o.Local.loadEnv(prefix)
	o.Log.loadEnv(prefix)
}

func (l *LocalConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "LOCAL")
	readEnv(prefix, "POLLING", &l.Polling, toBool)
	readEnv(prefix, "POLL_INTERVAL", &l.PollInterval, toInt)
	readEnvString(prefix, "FILE_PATH", &l.FilePath)
}

func (r *RedisConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "REDIS")
	readEnvString(prefix, "PASSWORD", &r.Password)
	readEnvString(prefix, "USER", &r.User)
	readEnv(prefix, "DB", &r.DB, toInt)
	readEnv(prefix, "ENABLED", &r.Enabled, toBool)
	readEnv(prefix, "ADDRESSES", &r.Addresses, toStringSlice)
	r.Tls.loadEnv(prefix)
}

func (s *SseConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "SSE")
	readEnv(prefix, "ENABLED", &s.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &s.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &s.Headers, toStringMap)
	readEnv(prefix, "HEARTBEAT_INTERVAL", &s.HeartBeatInterval, toInt)
	s.Log.loadEnv(prefix)
}

func (a *ApiConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "API")
	readEnv(prefix, "ENABLED", &a.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &a.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &a.Headers, toStringMap)
	readEnv(prefix, "AUTH_HEADERS", &a.AuthHeaders, toStringMap)
}

func (c *CdnProxyConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "CDN_PROXY")
	readEnv(prefix, "ENABLED", &c.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &c.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &c.Headers, toStringMap)
}

func (w *WebhookConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "WEBHOOK")
	readEnv(prefix, "ENABLED", &w.Enabled, toBool)
	readEnv(prefix, "AUTH_HEADERS", &w.AuthHeaders, toStringMap)
	w.Auth.loadEnv(prefix)
}

func (a *AuthConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "AUTH")
	readEnvString(prefix, "USER", &a.User)
	readEnvString(prefix, "PASSWORD", &a.Password)
}

func (t *TlsConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "TLS")
	readEnvString(prefix, "SERVER_NAME", &t.ServerName)
	readEnv(prefix, "MIN_VERSION", &t.MinVersion, toFloat)
	readEnv(prefix, "ENABLED", &t.Enabled, toBool)
	readEnv(prefix, "CERTIFICATES", &t.Certificates, toCertConfigSlice)
}

func (l *LogConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "LOG")
	readEnvString(prefix, "LEVEL", &l.Level)
}

func (m *MetricsConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "METRICS")
	readEnv(prefix, "ENABLED", &m.Enabled, toBool)
	readEnv(prefix, "PORT", &m.Port, toInt)
}

func readEnv[T any](prefix string, key string, in *T, conv func(string) (T, error)) {
	if env := os.Getenv(prefix + "_" + key); env != "" {
		if r, err := conv(env); err == nil {
			*in = r
		}
	}
}

func readEnvString(prefix string, key string, in *string) {
	if env := os.Getenv(prefix + "_" + key); env != "" {
		*in = env
	}
}

func concatPrefix(p1 string, p2 string) string {
	return p1 + "_" + p2
}
