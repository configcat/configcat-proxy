package config

import (
	"crypto/tls"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSDKConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_SDKS", `{"sdk1": "sdkKey"}`)
	t.Setenv("CONFIGCAT_SDK1_BASE_URL", "base")
	t.Setenv("CONFIGCAT_SDK1_POLL_INTERVAL", "300")
	t.Setenv("CONFIGCAT_SDK1_DATA_GOVERNANCE", "eu")
	t.Setenv("CONFIGCAT_SDK1_LOG_LEVEL", "error")

	t.Setenv("CONFIGCAT_SDK1_OFFLINE_ENABLED", "true")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_LOG_LEVEL", "debug")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_LOCAL_FILE_PATH", "./local.json")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_LOCAL_POLLING", "true")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_LOCAL_POLL_INTERVAL", "100")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_USE_CACHE", "true")
	t.Setenv("CONFIGCAT_SDK1_OFFLINE_CACHE_POLL_INTERVAL", "200")
	t.Setenv("CONFIGCAT_SDK1_WEBHOOK_SIGNING_KEY", "key")
	t.Setenv("CONFIGCAT_SDK1_WEBHOOK_SIGNATURE_VALID_FOR", "600")
	t.Setenv("CONFIGCAT_SDK1_DEFAULT_USER_ATTRIBUTES", `{"attr1": "attr_value1", "attr2": "attr_value2", "attr3": 5, "attr4":["a","b"]}`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "base", conf.SDKs["sdk1"].BaseUrl)
	assert.Equal(t, "sdkKey", conf.SDKs["sdk1"].Key)
	assert.Equal(t, 300, conf.SDKs["sdk1"].PollInterval)
	assert.Equal(t, "eu", conf.SDKs["sdk1"].DataGovernance)
	assert.Equal(t, log.Error, conf.SDKs["sdk1"].Log.GetLevel())

	assert.True(t, conf.SDKs["sdk1"].Offline.Enabled)
	assert.Equal(t, log.Debug, conf.SDKs["sdk1"].Offline.Log.GetLevel())
	assert.Equal(t, "./local.json", conf.SDKs["sdk1"].Offline.Local.FilePath)
	assert.True(t, conf.SDKs["sdk1"].Offline.Local.Polling)
	assert.Equal(t, 100, conf.SDKs["sdk1"].Offline.Local.PollInterval)
	assert.True(t, conf.SDKs["sdk1"].Offline.UseCache)
	assert.Equal(t, 200, conf.SDKs["sdk1"].Offline.CachePollInterval)
	assert.Equal(t, "key", conf.SDKs["sdk1"].WebhookSigningKey)
	assert.Equal(t, 600, conf.SDKs["sdk1"].WebhookSignatureValidFor)
	assert.Equal(t, "attr_value1", conf.SDKs["sdk1"].DefaultAttrs["attr1"])
	assert.Equal(t, "attr_value2", conf.SDKs["sdk1"].DefaultAttrs["attr2"])
	assert.Equal(t, float64(5), conf.SDKs["sdk1"].DefaultAttrs["attr3"])
	assert.Equal(t, []string{"a", "b"}, conf.SDKs["sdk1"].DefaultAttrs["attr4"])
}

func TestAutoSDKs_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_PROFILE_KEY", `key`)
	t.Setenv("CONFIGCAT_PROFILE_SECRET", `secret`)
	t.Setenv("CONFIGCAT_PROFILE_BASE_URL", `https://base.com`)
	t.Setenv("CONFIGCAT_PROFILE_SDKS_BASE_URL", `https://sdk-base.com`)
	t.Setenv("CONFIGCAT_PROFILE_SDKS_LOG_LEVEL", "info")
	t.Setenv("CONFIGCAT_PROFILE_POLL_INTERVAL", "300")
	t.Setenv("CONFIGCAT_PROFILE_WEBHOOK_SIGNING_KEY", "key")
	t.Setenv("CONFIGCAT_PROFILE_WEBHOOK_SIGNATURE_VALID_FOR", "600")
	t.Setenv("CONFIGCAT_PROFILE_LOG_LEVEL", "info")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "key", conf.Profile.Key)
	assert.Equal(t, "secret", conf.Profile.Secret)
	assert.Equal(t, "https://base.com", conf.Profile.BaseUrl)
	assert.Equal(t, "https://sdk-base.com", conf.Profile.SDKs.BaseUrl)
	assert.Equal(t, log.Info, conf.Profile.SDKs.Log.GetLevel())
	assert.Equal(t, 300, conf.Profile.PollInterval)
	assert.Equal(t, "key", conf.Profile.WebhookSigningKey)
	assert.Equal(t, 600, conf.Profile.WebhookSignatureValidFor)
	assert.Equal(t, log.Info, conf.Profile.Log.GetLevel())
}

func TestSDKConfig_ENV_Invalid(t *testing.T) {
	t.Run("sdk id", func(t *testing.T) {
		t.Setenv("CONFIGCAT_SDKS", `{"sdk1"}`)

		_, err := LoadConfigFromFileAndEnvironment("")
		require.ErrorContains(t, err, "failed to read environment variable 'CONFIGCAT_SDKS': invalid character '}' after object key")
	})
	t.Run("poll interval", func(t *testing.T) {
		t.Setenv("CONFIGCAT_SDKS", `{"sdk1": "sdkKey"}`)
		t.Setenv("CONFIGCAT_SDK1_POLL_INTERVAL", "a")

		_, err := LoadConfigFromFileAndEnvironment("")
		require.ErrorContains(t, err, `failed to read environment variable 'CONFIGCAT_SDK1_POLL_INTERVAL': strconv.Atoi: parsing "a": invalid syntax`)
	})
	t.Run("offline", func(t *testing.T) {
		t.Setenv("CONFIGCAT_SDKS", `{"sdk1": "sdkKey"}`)
		t.Setenv("CONFIGCAT_SDK1_OFFLINE_ENABLED", "b")

		_, err := LoadConfigFromFileAndEnvironment("")
		require.ErrorContains(t, err, `failed to read environment variable 'CONFIGCAT_SDK1_OFFLINE_ENABLED': strconv.ParseBool: parsing "b": invalid syntax`)
	})
	t.Run("user attrs", func(t *testing.T) {
		t.Setenv("CONFIGCAT_SDKS", `{"sdk1": "sdkKey"}`)
		t.Setenv("CONFIGCAT_SDK1_DEFAULT_USER_ATTRIBUTES", `{"attr1": true}`)

		_, err := LoadConfigFromFileAndEnvironment("")
		require.ErrorContains(t, err, `failed to read environment variable 'CONFIGCAT_SDK1_DEFAULT_USER_ATTRIBUTES': 'attr1' has an invalid type, only 'string', 'number', and 'string[]' types are allowed`)
	})
}

func TestRedisConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_CACHE_REDIS_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_REDIS_DB", "1")
	t.Setenv("CONFIGCAT_CACHE_REDIS_PASSWORD", "pass")
	t.Setenv("CONFIGCAT_CACHE_REDIS_USER", "user")
	t.Setenv("CONFIGCAT_CACHE_REDIS_ADDRESSES", `["addr1", "addr2"]`)
	t.Setenv("CONFIGCAT_CACHE_REDIS_TLS_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_REDIS_TLS_MIN_VERSION", "1.1")
	t.Setenv("CONFIGCAT_CACHE_REDIS_TLS_SERVER_NAME", "serv")
	t.Setenv("CONFIGCAT_CACHE_REDIS_TLS_CERTIFICATES", `[{"key":"./key1","cert":"./cert1"},{"key":"./key2","cert":"./cert2"}]`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.Cache.Redis.Enabled)
	assert.Equal(t, 1, conf.Cache.Redis.DB)
	assert.Equal(t, "pass", conf.Cache.Redis.Password)
	assert.Equal(t, "user", conf.Cache.Redis.User)
	assert.Equal(t, "addr1", conf.Cache.Redis.Addresses[0])
	assert.Equal(t, "addr2", conf.Cache.Redis.Addresses[1])
	assert.True(t, conf.Cache.Redis.Tls.Enabled)
	assert.Equal(t, tls.VersionTLS11, int(conf.Cache.Redis.Tls.GetVersion()))
	assert.Equal(t, "serv", conf.Cache.Redis.Tls.ServerName)
	assert.Equal(t, "./cert1", conf.Cache.Redis.Tls.Certificates[0].Cert)
	assert.Equal(t, "./key1", conf.Cache.Redis.Tls.Certificates[0].Key)
	assert.Equal(t, "./cert2", conf.Cache.Redis.Tls.Certificates[1].Cert)
	assert.Equal(t, "./key2", conf.Cache.Redis.Tls.Certificates[1].Key)
}

func TestMongoDbConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_CACHE_MONGODB_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_URL", "url")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_DATABASE", "db")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_COLLECTION", "coll")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_TLS_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_TLS_MIN_VERSION", "1.1")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_TLS_SERVER_NAME", "serv")
	t.Setenv("CONFIGCAT_CACHE_MONGODB_TLS_CERTIFICATES", `[{"key":"./key1","cert":"./cert1"},{"key":"./key2","cert":"./cert2"}]`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.Cache.MongoDb.Enabled)
	assert.Equal(t, "url", conf.Cache.MongoDb.Url)
	assert.Equal(t, "db", conf.Cache.MongoDb.Database)
	assert.Equal(t, "coll", conf.Cache.MongoDb.Collection)
	assert.True(t, conf.Cache.MongoDb.Tls.Enabled)
	assert.Equal(t, tls.VersionTLS11, int(conf.Cache.MongoDb.Tls.GetVersion()))
	assert.Equal(t, "serv", conf.Cache.MongoDb.Tls.ServerName)
	assert.Equal(t, "./cert1", conf.Cache.MongoDb.Tls.Certificates[0].Cert)
	assert.Equal(t, "./key1", conf.Cache.MongoDb.Tls.Certificates[0].Key)
	assert.Equal(t, "./cert2", conf.Cache.MongoDb.Tls.Certificates[1].Cert)
	assert.Equal(t, "./key2", conf.Cache.MongoDb.Tls.Certificates[1].Key)
}

func TestDynamoDbConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_CACHE_DYNAMODB_ENABLED", "true")
	t.Setenv("CONFIGCAT_CACHE_DYNAMODB_URL", "url")
	t.Setenv("CONFIGCAT_CACHE_DYNAMODB_TABLE", "db")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.Cache.DynamoDb.Enabled)
	assert.Equal(t, "url", conf.Cache.DynamoDb.Url)
	assert.Equal(t, "db", conf.Cache.DynamoDb.Table)
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

func TestDiagConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_DIAG_ENABLED", "false")
	t.Setenv("CONFIGCAT_DIAG_PORT", "8091")
	t.Setenv("CONFIGCAT_DIAG_METRICS_ENABLED", "false")
	t.Setenv("CONFIGCAT_DIAG_STATUS_ENABLED", "false")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.False(t, conf.Diag.Enabled)
	assert.Equal(t, 8091, conf.Diag.Port)
	assert.False(t, conf.Diag.Status.Enabled)
	assert.False(t, conf.Diag.Metrics.Enabled)
}

func TestGlobalOfflineConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_OFFLINE_ENABLED", "true")
	t.Setenv("CONFIGCAT_OFFLINE_CACHE_POLL_INTERVAL", "200")
	t.Setenv("CONFIGCAT_OFFLINE_LOG_LEVEL", "info")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.GlobalOfflineConfig.Enabled)
	assert.Equal(t, 200, conf.GlobalOfflineConfig.CachePollInterval)
	assert.Equal(t, log.Info, conf.GlobalOfflineConfig.Log.GetLevel())
}

func TestHttpConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_HTTP_PORT", "8090")
	t.Setenv("CONFIGCAT_HTTP_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_LOG_LEVEL", "info")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_USER", "mickey")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_PASSWORD", "pass")
	t.Setenv("CONFIGCAT_HTTP_WEBHOOK_AUTH_HEADERS", `{"X-API-KEY1": "auth1", "X-API-KEY2": "auth2"}`)
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_HEADERS", `{"CUSTOM-HEADER1": "cdn-val1", "CUSTOM-HEADER2": "cdn-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_SSE_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_LOG_LEVEL", "warn")
	t.Setenv("CONFIGCAT_HTTP_SSE_HEARTBEAT_INTERVAL", "5")
	t.Setenv("CONFIGCAT_HTTP_SSE_HEADERS", `{"CUSTOM-HEADER1": "sse-val1", "CUSTOM-HEADER2": "sse-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_API_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ALLOWED_ORIGINS", `["https://example1.com","https://example2.com"]`)
	t.Setenv("CONFIGCAT_HTTP_API_HEADERS", `{"CUSTOM-HEADER1": "api-val1", "CUSTOM-HEADER2": "api-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_API_AUTH_HEADERS", `{"X-API-KEY1": "api-auth1", "X-API-KEY2": "api-auth2"}`)
	t.Setenv("CONFIGCAT_HTTP_OFREP_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_OFREP_CORS_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_OFREP_CORS_ALLOWED_ORIGINS", `["https://example1.com","https://example2.com"]`)
	t.Setenv("CONFIGCAT_HTTP_OFREP_HEADERS", `{"CUSTOM-HEADER1": "ofrep-val1", "CUSTOM-HEADER2": "ofrep-val2"}`)
	t.Setenv("CONFIGCAT_HTTP_OFREP_AUTH_HEADERS", `{"X-API-KEY1": "ofrep-auth1", "X-API-KEY2": "ofrep-auth2"}`)
	t.Setenv("CONFIGCAT_HTTP_STATUS_ENABLED", "true")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, log.Info, conf.Http.Log.GetLevel())
	assert.Equal(t, 8090, conf.Http.Port)
	assert.True(t, conf.Http.Enabled)
	assert.True(t, conf.Http.Webhook.Enabled)
	assert.Equal(t, "mickey", conf.Http.Webhook.Auth.User)
	assert.Equal(t, "pass", conf.Http.Webhook.Auth.Password)
	assert.Equal(t, "auth1", conf.Http.Webhook.AuthHeaders["X-API-KEY1"])
	assert.Equal(t, "auth2", conf.Http.Webhook.AuthHeaders["X-API-KEY2"])

	assert.True(t, conf.Http.CdnProxy.Enabled)
	assert.True(t, conf.Http.CdnProxy.CORS.Enabled)
	assert.Nil(t, conf.Http.CdnProxy.CORS.AllowedOrigins)
	assert.Equal(t, "cdn-val1", conf.Http.CdnProxy.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "cdn-val2", conf.Http.CdnProxy.Headers["CUSTOM-HEADER2"])

	assert.True(t, conf.Http.Sse.Enabled)
	assert.True(t, conf.Http.Sse.CORS.Enabled)
	assert.Nil(t, conf.Http.Sse.CORS.AllowedOrigins)
	assert.Equal(t, log.Warn, conf.Http.Sse.Log.GetLevel())
	assert.Equal(t, "sse-val1", conf.Http.Sse.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "sse-val2", conf.Http.Sse.Headers["CUSTOM-HEADER2"])
	assert.Equal(t, 5, conf.Http.Sse.HeartBeatInterval)

	assert.True(t, conf.Http.Api.Enabled)
	assert.True(t, conf.Http.Api.CORS.Enabled)
	assert.Equal(t, "https://example1.com", conf.Http.Api.CORS.AllowedOrigins[0])
	assert.Equal(t, "https://example2.com", conf.Http.Api.CORS.AllowedOrigins[1])
	assert.Equal(t, "api-val1", conf.Http.Api.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "api-val2", conf.Http.Api.Headers["CUSTOM-HEADER2"])
	assert.Equal(t, "api-auth1", conf.Http.Api.AuthHeaders["X-API-KEY1"])
	assert.Equal(t, "api-auth2", conf.Http.Api.AuthHeaders["X-API-KEY2"])

	assert.True(t, conf.Http.OFREP.Enabled)
	assert.True(t, conf.Http.OFREP.CORS.Enabled)
	assert.Equal(t, "https://example1.com", conf.Http.OFREP.CORS.AllowedOrigins[0])
	assert.Equal(t, "https://example2.com", conf.Http.OFREP.CORS.AllowedOrigins[1])
	assert.Equal(t, "ofrep-val1", conf.Http.OFREP.Headers["CUSTOM-HEADER1"])
	assert.Equal(t, "ofrep-val2", conf.Http.OFREP.Headers["CUSTOM-HEADER2"])
	assert.Equal(t, "ofrep-auth1", conf.Http.OFREP.AuthHeaders["X-API-KEY1"])
	assert.Equal(t, "ofrep-auth2", conf.Http.OFREP.AuthHeaders["X-API-KEY2"])

	assert.True(t, conf.Http.Status.Enabled)
}

func TestCORSConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_HTTP_CDN_PROXY_CORS_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_LOG_LEVEL", "warn")
	t.Setenv("CONFIGCAT_HTTP_SSE_ALLOW_CORS", "true")
	t.Setenv("CONFIGCAT_HTTP_SSE_HEARTBEAT_INTERVAL", "5")
	t.Setenv("CONFIGCAT_HTTP_SSE_CORS_ALLOWED_ORIGINS", `["https://example1.com","https://example2.com"]`)
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ENABLED", "true")
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ALLOWED_ORIGINS", `["https://example1.com","https://example2.com"]`)
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ALLOWED_ORIGINS_REGEX_PATTERNS", `[".*\\.example1\\.com",".*\\.example2\\.com"]`)
	t.Setenv("CONFIGCAT_HTTP_API_CORS_ALLOWED_ORIGINS_REGEX_IF_NO_MATCH", "https://example1.com")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.True(t, conf.Http.CdnProxy.CORS.Enabled)
	assert.Nil(t, conf.Http.CdnProxy.CORS.AllowedOrigins)

	assert.True(t, conf.Http.Sse.CORS.Enabled)
	assert.Equal(t, "https://example1.com", conf.Http.Sse.CORS.AllowedOrigins[0])
	assert.Equal(t, "https://example2.com", conf.Http.Sse.CORS.AllowedOrigins[1])
	assert.Nil(t, conf.Http.Sse.CORS.AllowedOriginsRegex.Patterns)
	assert.Equal(t, "", conf.Http.Sse.CORS.AllowedOriginsRegex.IfNoMatch)

	assert.True(t, conf.Http.Api.CORS.Enabled)
	assert.Equal(t, "https://example1.com", conf.Http.Api.CORS.AllowedOrigins[0])
	assert.Equal(t, "https://example2.com", conf.Http.Api.CORS.AllowedOrigins[1])
	assert.Equal(t, `.*\.example1\.com`, conf.Http.Api.CORS.AllowedOriginsRegex.Patterns[0])
	assert.Equal(t, `.*\.example2\.com`, conf.Http.Api.CORS.AllowedOriginsRegex.Patterns[1])
	assert.Equal(t, "https://example1.com", conf.Http.Api.CORS.AllowedOriginsRegex.IfNoMatch)
}

func TestGrpcConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_GRPC_PORT", "8060")
	t.Setenv("CONFIGCAT_GRPC_LOG_LEVEL", "error")
	t.Setenv("CONFIGCAT_GRPC_ENABLED", "true")
	t.Setenv("CONFIGCAT_GRPC_HEALTH_CHECK_ENABLED", "false")
	t.Setenv("CONFIGCAT_GRPC_SERVER_REFLECTION_ENABLED", "true")
	t.Setenv("CONFIGCAT_GRPC_KEEP_ALIVE_MAX_CONNECTION_IDLE", "1")
	t.Setenv("CONFIGCAT_GRPC_KEEP_ALIVE_MAX_CONNECTION_AGE", "2")
	t.Setenv("CONFIGCAT_GRPC_KEEP_ALIVE_MAX_CONNECTION_AGE_GRACE", "3")
	t.Setenv("CONFIGCAT_GRPC_KEEP_ALIVE_TIME", "4")
	t.Setenv("CONFIGCAT_GRPC_KEEP_ALIVE_TIMEOUT", "5")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, log.Error, conf.Grpc.Log.GetLevel())
	assert.Equal(t, 8060, conf.Grpc.Port)
	assert.True(t, conf.Grpc.Enabled)
	assert.True(t, conf.Grpc.ServerReflectionEnabled)
	assert.False(t, conf.Grpc.HealthCheckEnabled)
	assert.Equal(t, 1, conf.Grpc.KeepAlive.MaxConnectionIdle)
	assert.Equal(t, 2, conf.Grpc.KeepAlive.MaxConnectionAge)
	assert.Equal(t, 3, conf.Grpc.KeepAlive.MaxConnectionAgeGrace)
	assert.Equal(t, 4, conf.Grpc.KeepAlive.Time)
	assert.Equal(t, 5, conf.Grpc.KeepAlive.Timeout)
}

func TestHttpProxyConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_HTTP_PROXY_URL", "proxy-url")

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "proxy-url", conf.HttpProxy.Url)
}

func TestGlobalDefaultAttributesConfig_ENV(t *testing.T) {
	t.Setenv("CONFIGCAT_LOG_LEVEL", "error")
	t.Setenv("CONFIGCAT_DEFAULT_USER_ATTRIBUTES", `{"attr_1": "attr_value1", "attr2": "attr_value2"}`)

	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, "attr_value1", conf.DefaultAttrs["attr_1"])
	assert.Equal(t, "attr_value2", conf.DefaultAttrs["attr2"])
}
