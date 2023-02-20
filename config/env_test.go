package config

import (
	"crypto/tls"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSDKConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_SDK_BASE_URL", "base")
	t.Setenv("CONFIGCAT_SDK_KEY", "sdkKey")
	t.Setenv("CONFIGCAT_SDK_POLL_INTERVAL", "300")
	t.Setenv("CONFIGCAT_SDK_DATA_GOVERNANCE", "eu")
	t.Setenv("CONFIGCAT_SDK_LOG_LEVEL", "error")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_ENABLED", "true")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_LOG_LEVEL", "debug")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_LOCAL_FILE_PATH", "./local.json")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_LOCAL_POLLING", "true")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_LOCAL_POLL_INTERVAL", "100")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_USE_CACHE", "true")
	t.Setenv("CONFIGCAT_SDK_OFFLINE_CACHE_POLL_INTERVAL", "200")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_ENABLED", "true")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_DB", "1")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_PASSWORD", "pass")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_ADDRESSES", `["addr1", "addr2"]`)
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_TLS_ENABLED", "true")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_TLS_MIN_VERSION", "1.1")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_TLS_SERVER_NAME", "serv")
	t.Setenv("CONFIGCAT_SDK_CACHE_REDIS_TLS_CERTIFICATES", `[{"key":"./key1","cert":"./cert1"},{"key":"./key2","cert":"./cert2"}]`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "base", conf.SDK.BaseUrl)
	assert.Equal(t, "sdkKey", conf.SDK.Key)
	assert.Equal(t, 300, conf.SDK.PollInterval)
	assert.Equal(t, "eu", conf.SDK.DataGovernance)
	assert.Equal(t, log.Error, conf.SDK.Log.GetLevel())

	assert.True(t, conf.SDK.Offline.Enabled)
	assert.Equal(t, log.Debug, conf.SDK.Offline.Log.GetLevel())
	assert.Equal(t, "./local.json", conf.SDK.Offline.Local.FilePath)
	assert.True(t, conf.SDK.Offline.Local.Polling)
	assert.Equal(t, 100, conf.SDK.Offline.Local.PollInterval)
	assert.True(t, conf.SDK.Offline.UseCache)
	assert.Equal(t, 200, conf.SDK.Offline.CachePollInterval)

	assert.True(t, conf.SDK.Cache.Redis.Enabled)
	assert.Equal(t, 1, conf.SDK.Cache.Redis.DB)
	assert.Equal(t, "pass", conf.SDK.Cache.Redis.Password)
	assert.Equal(t, "addr1", conf.SDK.Cache.Redis.Addresses[0])
	assert.Equal(t, "addr2", conf.SDK.Cache.Redis.Addresses[1])
	assert.True(t, conf.SDK.Cache.Redis.Tls.Enabled)
	assert.Equal(t, tls.VersionTLS11, int(conf.SDK.Cache.Redis.Tls.GetVersion()))
	assert.Equal(t, "serv", conf.SDK.Cache.Redis.Tls.ServerName)
	assert.Equal(t, "./cert1", conf.SDK.Cache.Redis.Tls.Certificates[0].Cert)
	assert.Equal(t, "./key1", conf.SDK.Cache.Redis.Tls.Certificates[0].Key)
	assert.Equal(t, "./cert2", conf.SDK.Cache.Redis.Tls.Certificates[1].Cert)
	assert.Equal(t, "./key2", conf.SDK.Cache.Redis.Tls.Certificates[1].Key)
}

func TestTlsConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_TLS_ENABLED", "true")
	t.Setenv("CONFIGCAT_TLS_MIN_VERSION", "1.1")
	t.Setenv("CONFIGCAT_TLS_SERVER_NAME", "serv")
	t.Setenv("CONFIGCAT_TLS_CERTIFICATES", `[{"key":"./key1","cert":"./cert1"},{"key":"./key2","cert":"./cert2"}]`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.Tls.Enabled)
	assert.Equal(t, tls.VersionTLS11, int(conf.Tls.GetVersion()))
	assert.Equal(t, "serv", conf.Tls.ServerName)
	assert.Equal(t, "./cert1", conf.Tls.Certificates[0].Cert)
	assert.Equal(t, "./key1", conf.Tls.Certificates[0].Key)
	assert.Equal(t, "./cert2", conf.Tls.Certificates[1].Cert)
	assert.Equal(t, "./key2", conf.Tls.Certificates[1].Key)
}

func TestLogConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_LOG_LEVEL", "error")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, log.Error, conf.Log.GetLevel())
}

func TestMetricsConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_METRICS_ENABLED", "false")
	t.Setenv("CONFIGCAT_METRICS_PORT", "8091")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.False(t, conf.Metrics.Enabled)
	assert.Equal(t, 8091, conf.Metrics.Port)
}

func TestHttpConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_HTTP_PORT", "8090")
	t.Setenv("CONFIGCAT_HTTP_LOG_LEVEL", "info")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_SIGNING_KEY", "key")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_SIGNATURE_VALID_FOR", "600")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_USER", "mickey")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_PASSWORD", "pass")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_HEADERS", `{"X-API-KEY1": "auth1", "X-API-KEY2": "auth2"}`)
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_HEADERS", `{"CUSTOM-HEADER1": "cdn-val1", "CUSTOM-HEADER2": "cdn-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_ALLOW_CORS", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_LOG_LEVEL", "warn")
	t.Setenv("CONFIGCAT_HTTP_SSE_ALLOW_CORS", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_HEARTBEAT_INTERVAL", "5")
	t.Setenv("CONFIGCAT_HTTP_SSE_HEADERS", `{"CUSTOM-HEADER1": "sse-val1", "CUSTOM-HEADER2": "sse-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_API_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_API_ALLOW_CORS", "true")
	t.Setenv("CONFIGCAT_HTTP_API_HEADERS", `{"CUSTOM-HEADER1": "api-val1", "CUSTOM-HEADER2": "api-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_API_AUTH_HEADERS", `{"X-API-KEY1": "api-auth1", "X-API-KEY2": "api-auth2"}`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, log.Info, conf.Http.Log.GetLevel())
	assert.Equal(t, 8090, conf.Http.Port)
	assert.True(t, conf.Http.Webhook.Enabled)
	assert.Equal(t, "key", conf.Http.Webhook.SigningKey)
	assert.Equal(t, 600, conf.Http.Webhook.SignatureValidFor)
	assert.Equal(t, "mickey", conf.Http.Webhook.Auth.User)
	assert.Equal(t, "pass", conf.Http.Webhook.Auth.Password)
	assert.Equal(t, "auth1", conf.Http.Webhook.AuthHeaders["X-API-KEY1"])
	assert.Equal(t, "auth2", conf.Http.Webhook.AuthHeaders["X-API-KEY2"])

	assert.True(t, conf.Http.CdnProxy.Enabled)
	assert.True(t, conf.Http.CdnProxy.AllowCORS)
	assert.Equal(t, "cdn-val1", conf.Http.CdnProxy.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "cdn-val2", conf.Http.CdnProxy.Headers["CUSTOM-HEADER2"])

	assert.True(t, conf.Http.Sse.Enabled)
	assert.True(t, conf.Http.Sse.AllowCORS)
	assert.Equal(t, log.Warn, conf.Http.Sse.Log.GetLevel())
	assert.Equal(t, "sse-val1", conf.Http.Sse.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "sse-val2", conf.Http.Sse.Headers["CUSTOM-HEADER2"])
	assert.Equal(t, 5, conf.Http.Sse.HeartBeatInterval)

	assert.True(t, conf.Http.Api.Enabled)
	assert.True(t, conf.Http.Api.AllowCORS)
	assert.Equal(t, "api-val1", conf.Http.Api.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "api-val2", conf.Http.Api.Headers["CUSTOM-HEADER2"])
	assert.Equal(t, "api-auth1", conf.Http.Api.AuthHeaders["X-API-KEY1"])
	assert.Equal(t, "api-auth2", conf.Http.Api.AuthHeaders["X-API-KEY2"])
}

func TestGrpcConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_GRPC_PORT", "8060")
	t.Setenv("CONFIGCAT_GRPC_LOG_LEVEL", "error")
	t.Setenv("CONFIGCAT_GRPC_ENABLED", "true")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, log.Error, conf.Grpc.Log.GetLevel())
	assert.Equal(t, 8060, conf.Grpc.Port)
	assert.True(t, conf.Grpc.Enabled)
}

func TestHttpProxyConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_HTTP_PROXY_URL", "proxy-url")
	t.Setenv("CONFIGCAT_HTTP_PROXY_USER", "proxy-user")
	t.Setenv("CONFIGCAT_HTTP_PROXY_PASSWORD", "proxy-pass")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "proxy-url", conf.HttpProxy.Url)
	assert.Equal(t, "proxy-user", conf.HttpProxy.User)
	assert.Equal(t, "proxy-pass", conf.HttpProxy.Password)
}