package config

import (
	"crypto/tls"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfig_Defaults(t *testing.T) {
	conf, err := LoadConfigFromFileAndEnvironment("")
	require.NoError(t, err)

	assert.Equal(t, 8050, conf.Http.Port)

	assert.Equal(t, 50051, conf.Grpc.Port)
	assert.True(t, conf.Grpc.Enabled)

	assert.Equal(t, 8051, conf.Metrics.Port)
	assert.True(t, conf.Metrics.Enabled)

	assert.True(t, conf.Http.Sse.Enabled)
	assert.True(t, conf.Http.Sse.AllowCORS)

	assert.True(t, conf.Http.CdnProxy.Enabled)
	assert.True(t, conf.Http.CdnProxy.AllowCORS)

	assert.True(t, conf.Http.Api.Enabled)
	assert.True(t, conf.Http.Api.AllowCORS)

	assert.True(t, conf.Http.Webhook.Enabled)

	assert.False(t, conf.GlobalOfflineConfig.Enabled)
	assert.Equal(t, 5, conf.GlobalOfflineConfig.CachePollInterval)

	assert.Equal(t, 0, conf.Cache.Redis.DB)
	assert.Equal(t, "localhost:6379", conf.Cache.Redis.Addresses[0])

	assert.Equal(t, 1.2, conf.Tls.MinVersion)
	assert.Equal(t, 1.2, conf.Cache.Redis.Tls.MinVersion)

	assert.Equal(t, uint16(tls.VersionTLS12), conf.Tls.GetVersion())
	assert.Equal(t, uint16(tls.VersionTLS12), conf.Cache.Redis.Tls.GetVersion())

	assert.Nil(t, conf.DefaultAttrs)
}

func TestConfig_DefaultAttrs(t *testing.T) {
	utils.UseTempFile(`
sdks:
  test_sdk:
    key: key
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Nil(t, conf.SDKs["test_sdk"].DefaultAttrs)
	})
}

func TestConfig_LogLevelFixup(t *testing.T) {
	t.Run("valid base level", func(t *testing.T) {
		utils.UseTempFile(`
sdks:
  test_sdk:
    key: key
log:
  level: "info"
`, func(file string) {
			conf, err := LoadConfigFromFileAndEnvironment(file)
			require.NoError(t, err)

			assert.Equal(t, log.Info, conf.Log.GetLevel())
			assert.Equal(t, log.Info, conf.SDKs["test_sdk"].Log.GetLevel())
			assert.Equal(t, log.Info, conf.SDKs["test_sdk"].Offline.Log.GetLevel())
			assert.Equal(t, log.Info, conf.Http.Log.GetLevel())
			assert.Equal(t, log.Info, conf.Http.Sse.Log.GetLevel())
			assert.Equal(t, log.Info, conf.Grpc.Log.GetLevel())
			assert.Equal(t, log.Info, conf.GlobalOfflineConfig.Log.GetLevel())
		})
	})

	t.Run("invalid base level", func(t *testing.T) {
		utils.UseTempFile(`
sdks:
  test_sdk:
    key: key
log:
  level: "invalid"
`, func(file string) {
			conf, err := LoadConfigFromFileAndEnvironment(file)
			require.NoError(t, err)

			assert.Equal(t, log.Warn, conf.Log.GetLevel())
			assert.Equal(t, log.Warn, conf.SDKs["test_sdk"].Log.GetLevel())
			assert.Equal(t, log.Warn, conf.SDKs["test_sdk"].Offline.Log.GetLevel())
			assert.Equal(t, log.Warn, conf.Http.Log.GetLevel())
			assert.Equal(t, log.Warn, conf.Http.Sse.Log.GetLevel())
			assert.Equal(t, log.Warn, conf.Grpc.Log.GetLevel())
			assert.Equal(t, log.Warn, conf.GlobalOfflineConfig.Log.GetLevel())
		})
	})

	t.Run("overrides", func(t *testing.T) {
		utils.UseTempFile(`
log:
  level: "error"
sdks:
  test_sdk:
    log:
      level: "debug"
    offline: 
      log:
        level: "debug"
http:
  log:
    level: "debug"
  sse: 
    log:
      level: "debug"
grpc:
  log:
    level: "debug"

offline:
  log:
    level: "debug"
`, func(file string) {
			conf, err := LoadConfigFromFileAndEnvironment(file)
			require.NoError(t, err)

			assert.Equal(t, log.Error, conf.Log.GetLevel())
			assert.Equal(t, log.Debug, conf.SDKs["test_sdk"].Log.GetLevel())
			assert.Equal(t, log.Debug, conf.SDKs["test_sdk"].Offline.Log.GetLevel())
			assert.Equal(t, log.Debug, conf.Http.Log.GetLevel())
			assert.Equal(t, log.Debug, conf.Http.Sse.Log.GetLevel())
			assert.Equal(t, log.Debug, conf.Grpc.Log.GetLevel())
			assert.Equal(t, log.Debug, conf.GlobalOfflineConfig.Log.GetLevel())
		})
	})
}

func TestSDKConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
sdks:
  test_sdk:
    base_url: "base"
    key: "sdkKey"
    poll_interval: 300
    data_governance: "eu"
    webhook_signing_key: "key"
    webhook_signature_valid_for: 600
    default_user_attributes:
      attr_1: "attr_value1"
      attr2: "attr_value2"
      attr3: 
      attr 4: "attr value4"
    log:
      level: "error"
    offline:
      enabled: true
      log:
        level: "debug"
      local:
        file_path: "./local.json"
        polling: true
        poll_interval: 100
      use_cache: true
      cache_poll_interval: 200
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, "base", conf.SDKs["test_sdk"].BaseUrl)
		assert.Equal(t, "sdkKey", conf.SDKs["test_sdk"].Key)
		assert.Equal(t, 300, conf.SDKs["test_sdk"].PollInterval)
		assert.Equal(t, "eu", conf.SDKs["test_sdk"].DataGovernance)
		assert.Equal(t, log.Error, conf.SDKs["test_sdk"].Log.GetLevel())
		assert.Equal(t, "key", conf.SDKs["test_sdk"].WebhookSigningKey)
		assert.Equal(t, 600, conf.SDKs["test_sdk"].WebhookSignatureValidFor)

		assert.True(t, conf.SDKs["test_sdk"].Offline.Enabled)
		assert.Equal(t, log.Debug, conf.SDKs["test_sdk"].Offline.Log.GetLevel())
		assert.Equal(t, "./local.json", conf.SDKs["test_sdk"].Offline.Local.FilePath)
		assert.True(t, conf.SDKs["test_sdk"].Offline.Local.Polling)
		assert.Equal(t, 100, conf.SDKs["test_sdk"].Offline.Local.PollInterval)
		assert.True(t, conf.SDKs["test_sdk"].Offline.UseCache)
		assert.Equal(t, 200, conf.SDKs["test_sdk"].Offline.CachePollInterval)
		assert.Equal(t, 200, conf.SDKs["test_sdk"].Offline.CachePollInterval)

		assert.Equal(t, "attr_value1", conf.SDKs["test_sdk"].DefaultAttrs["attr_1"])
		assert.Equal(t, "attr_value2", conf.SDKs["test_sdk"].DefaultAttrs["attr2"])
		assert.Equal(t, "", conf.SDKs["test_sdk"].DefaultAttrs["attr3"])
		assert.Equal(t, "attr value4", conf.SDKs["test_sdk"].DefaultAttrs["attr 4"])
	})
}

func TestSDKWithGlobalOffline_YAML(t *testing.T) {
	utils.UseTempFile(`
sdks:
  test_sdk_1:
    poll_interval: 30
    base_url: "test"
    key: "sdkKey1"
  test_sdk_2:
    key: "sdkKey2"
    offline:
      enabled: true
      local:
        file_path: "./local.json"
  test_sdk_3:
    key: "sdkKey3"
    offline:
      enabled: true
      use_cache: true
      cache_poll_interval: 20
      

offline:
  enabled: true
  cache_poll_interval: 10
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.True(t, conf.SDKs["test_sdk_1"].Offline.Enabled)
		assert.True(t, conf.SDKs["test_sdk_1"].Offline.UseCache)
		assert.Equal(t, 10, conf.SDKs["test_sdk_1"].Offline.CachePollInterval)

		assert.True(t, conf.SDKs["test_sdk_2"].Offline.Enabled)
		assert.False(t, conf.SDKs["test_sdk_2"].Offline.UseCache)
		assert.Equal(t, "./local.json", conf.SDKs["test_sdk_2"].Offline.Local.FilePath)

		assert.True(t, conf.SDKs["test_sdk_3"].Offline.Enabled)
		assert.True(t, conf.SDKs["test_sdk_3"].Offline.UseCache)
		assert.Equal(t, 20, conf.SDKs["test_sdk_3"].Offline.CachePollInterval)
	})
}

func TestSDKWithGlobalOfflineAndEnv_YAML(t *testing.T) {
	utils.UseTempFile(`
sdks:
  test_sdk_1:
    poll_interval: 30
    base_url: "test"
    key: "sdkKey1"
  test_sdk_2:
    key: "sdkKey2"
    offline:
      enabled: true
      local:
        file_path: "./local.json"
  test_sdk_3:
    key: "sdkKey3"
    offline:
      enabled: true
      use_cache: true
      cache_poll_interval: 20
`, func(file string) {
		t.Setenv("CONFIGCAT_OFFLINE_ENABLED", "true")
		t.Setenv("CONFIGCAT_OFFLINE_CACHE_POLL_INTERVAL", "10")

		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.True(t, conf.SDKs["test_sdk_1"].Offline.Enabled)
		assert.True(t, conf.SDKs["test_sdk_1"].Offline.UseCache)
		assert.Equal(t, 10, conf.SDKs["test_sdk_1"].Offline.CachePollInterval)

		assert.True(t, conf.SDKs["test_sdk_2"].Offline.Enabled)
		assert.False(t, conf.SDKs["test_sdk_2"].Offline.UseCache)
		assert.Equal(t, "./local.json", conf.SDKs["test_sdk_2"].Offline.Local.FilePath)

		assert.True(t, conf.SDKs["test_sdk_3"].Offline.Enabled)
		assert.True(t, conf.SDKs["test_sdk_3"].Offline.UseCache)
		assert.Equal(t, 20, conf.SDKs["test_sdk_3"].Offline.CachePollInterval)
	})
}

func TestCacheConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
cache:
  redis:
    enabled: true
    db: 1
    password: "pass"
    user: "user"
    addresses: ["addr1", "addr2"]
    tls: 
      enabled: true
      min_version: 1.1
      server_name: "serv"
      certificates:
        - cert: "./cert1"
          key: "./key1"
        - cert: "./cert2"
          key: "./key2"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
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
	})
}

func TestGlobalOfflineConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
offline:
  enabled: true
  cache_poll_interval: 200
  log:
    level: "error"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.True(t, conf.GlobalOfflineConfig.Enabled)
		assert.Equal(t, 200, conf.GlobalOfflineConfig.CachePollInterval)
		assert.Equal(t, log.Error, conf.GlobalOfflineConfig.Log.GetLevel())
	})
}

func TestTlsConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
tls: 
  enabled: true
  min_version: 1.1
  server_name: "serv"
  certificates:
    - cert: "./cert1"
      key: "./key1"
    - cert: "./cert2"
      key: "./key2"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.True(t, conf.Tls.Enabled)
		assert.Equal(t, tls.VersionTLS11, int(conf.Tls.GetVersion()))
		assert.Equal(t, "serv", conf.Tls.ServerName)
		assert.Equal(t, "./cert1", conf.Tls.Certificates[0].Cert)
		assert.Equal(t, "./key1", conf.Tls.Certificates[0].Key)
		assert.Equal(t, "./cert2", conf.Tls.Certificates[1].Cert)
		assert.Equal(t, "./key2", conf.Tls.Certificates[1].Key)
	})
}

func TestLogConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
log:
  level: "error"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, log.Error, conf.Log.GetLevel())
	})
}

func TestMetricsConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
metrics:
  enabled: false
  port: 8091
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.False(t, conf.Metrics.Enabled)
		assert.Equal(t, 8091, conf.Metrics.Port)
	})
}

func TestHttpConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
http:
  port: 8090
  log: 
    level: "info"
  webhook:
    enabled: true
    auth:
      user: "mickey"
      password: "pass"
    auth_headers:
      X-API-KEY1: "auth1"
      X-API-KEY2: "auth2"
  cdn_proxy:
    enabled: true
    headers:
      CUSTOM-HEADER1: "cdn-val1"
      CUSTOM-HEADER2: "cdn-val2"
    allow_cors: true
  api:
    enabled: true
    allow_cors: true
    headers:
      CUSTOM-HEADER1: "api-val1"
      CUSTOM-HEADER2: "api-val2"
    auth_headers:
      X-API-KEY1: "api-auth1"
      X-API-KEY2: "api-auth2"
  sse:
    log: 
      level: "warn"
    enabled: true
    allow_cors: true
    heart_beat_interval: 5
    headers:
      CUSTOM-HEADER1: "sse-val1"
      CUSTOM-HEADER2: "sse-val2"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, log.Info, conf.Http.Log.GetLevel())
		assert.Equal(t, 8090, conf.Http.Port)
		assert.True(t, conf.Http.Webhook.Enabled)
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
	})
}

func TestGrpcConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
grpc:
  enabled: true
  port: 8060
  log:
    level: "error"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, log.Error, conf.Grpc.Log.GetLevel())
		assert.Equal(t, 8060, conf.Grpc.Port)
		assert.True(t, conf.Grpc.Enabled)
	})
}

func TestHttpProxyConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
http_proxy:
  url: "proxy-url"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, "proxy-url", conf.HttpProxy.Url)
	})
}

func TestDefaultAttributesConfig_YAML(t *testing.T) {
	utils.UseTempFile(`
default_user_attributes:
  attr_1: "attr_value1"
  attr2: "attr_value2"
  attr3:
  attr 4: "attr value4"
`, func(file string) {
		conf, err := LoadConfigFromFileAndEnvironment(file)
		require.NoError(t, err)

		assert.Equal(t, "attr_value1", conf.DefaultAttrs["attr_1"])
		assert.Equal(t, "attr_value2", conf.DefaultAttrs["attr2"])
		assert.Equal(t, "", conf.DefaultAttrs["attr3"])
		assert.Equal(t, "attr value4", conf.DefaultAttrs["attr 4"])
	})
}
