package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
var toHeaderMap = func(s string) (map[string]string, error) {
	var r map[string]string
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Config) loadEnv() {
	c.SDK.loadEnv(envPrefix)
	c.Http.loadEnv(envPrefix)
	c.Grpc.loadEnv(envPrefix)
	c.HttpProxy.loadEnv(envPrefix)
	c.Log.loadEnv(envPrefix)
	c.Tls.loadEnv(envPrefix)
	c.Metrics.loadEnv(envPrefix)
}

func (s *SDKConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "SDK")
	readEnvString(prefix, "KEY", &s.Key)
	readEnvString(prefix, "BASE_URL", &s.BaseUrl)
	readEnvString(prefix, "DATA_GOVERNANCE", &s.DataGovernance)
	readEnv(prefix, "POLL_INTERVAL", &s.PollInterval, toInt)
	s.Cache.loadEnv(prefix)
	s.Offline.loadEnv(prefix)
	s.EvalStats.loadEnv(prefix)
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

func (e *EvalStatsConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "EVAL_STATS")
	e.InfluxDb.loadEnv(prefix)
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
	readEnv(prefix, "DB", &r.DB, toInt)
	readEnv(prefix, "ENABLED", &r.Enabled, toBool)
	readEnv(prefix, "ADDRESSES", &r.Addresses, toStringSlice)
	r.Tls.loadEnv(prefix)
}

func (i *InfluxDbConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "INFLUX_DB")
	readEnvString(prefix, "URL", &i.Url)
	readEnvString(prefix, "ORGANIZATION", &i.Organization)
	readEnvString(prefix, "AUTH_TOKEN", &i.AuthToken)
	readEnvString(prefix, "BUCKET", &i.Bucket)
	readEnv(prefix, "ENABLED", &i.Enabled, toBool)
	i.Tls.loadEnv(prefix)
}

func (s *SseConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "SSE")
	readEnv(prefix, "ENABLED", &s.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &s.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &s.Headers, toHeaderMap)
	readEnv(prefix, "HEARTBEAT_INTERVAL", &s.HeartBeatInterval, toInt)
	s.Log.loadEnv(prefix)
}

func (a *ApiConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "API")
	readEnv(prefix, "ENABLED", &a.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &a.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &a.Headers, toHeaderMap)
	readEnv(prefix, "AUTH_HEADERS", &a.AuthHeaders, toHeaderMap)
}

func (c *CdnProxyConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "CDN_PROXY")
	readEnv(prefix, "ENABLED", &c.Enabled, toBool)
	readEnv(prefix, "ALLOW_CORS", &c.AllowCORS, toBool)
	readEnv(prefix, "HEADERS", &c.Headers, toHeaderMap)
}

func (w *WebhookConfig) loadEnv(prefix string) {
	prefix = concatPrefix(prefix, "WEBHOOK")
	readEnvString(prefix, "SIGNING_KEY", &w.SigningKey)
	readEnv(prefix, "ENABLED", &w.Enabled, toBool)
	readEnv(prefix, "SIGNATURE_VALID_FOR", &w.SignatureValidFor, toInt)
	readEnv(prefix, "AUTH_HEADERS", &w.AuthHeaders, toHeaderMap)
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
	if env := os.Getenv(fmt.Sprintf("%s_%s", prefix, key)); env != "" {
		if r, err := conv(env); err == nil {
			*in = r
		}
	}
}

func readEnvString(prefix string, key string, in *string) {
	if env := os.Getenv(fmt.Sprintf("%s_%s", prefix, key)); env != "" {
		*in = env
	}
}

func concatPrefix(p1 string, p2 string) string {
	return fmt.Sprintf("%s_%s", p1, p2)
}
