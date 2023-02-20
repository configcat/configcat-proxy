package config

import (
	"github.com/configcat/configcat-proxy/utils"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("invalid sdk key", func(t *testing.T) {
		conf := Config{}
		require.ErrorContains(t, conf.Validate(), "sdk: SDK key is required")
	})
	t.Run("invalid data governance", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", DataGovernance: "inv"}}
		require.ErrorContains(t, conf.Validate(), "sdk: invalid data governance value, it must be 'global' or 'eu'")
	})
	t.Run("offline invalid file path", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Offline: OfflineConfig{Enabled: true, Local: LocalConfig{FilePath: "nonexisting"}}}}
		require.ErrorContains(t, conf.Validate(), "sdk: couldn't find the local file")
	})
	t.Run("offline file polling invalid poll interval", func(t *testing.T) {
		utils.UseTempFile("", func(path string) {
			conf := Config{SDK: SDKConfig{Key: "Key", Offline: OfflineConfig{Enabled: true, Local: LocalConfig{FilePath: path, Polling: true, PollInterval: 0}}}}
			require.ErrorContains(t, conf.Validate(), "sdk: local file poll interval must be greater than 1 seconds")
		})
	})
	t.Run("offline enabled without file and cache", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Offline: OfflineConfig{Enabled: true}}}
		require.ErrorContains(t, conf.Validate(), "sdk: offline mode requires either a configured cache or a local file")
	})
	t.Run("offline cache without redis", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Offline: OfflineConfig{Enabled: true, UseCache: true}}}
		require.ErrorContains(t, conf.Validate(), "sdk: offline mode enabled with cache, but no cache is configured")
	})
	t.Run("offline cache invalid poll interval", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Offline: OfflineConfig{Enabled: true, UseCache: true, CachePollInterval: 0}, Cache: CacheConfig{Redis: RedisConfig{}}}}
		require.ErrorContains(t, conf.Validate(), "sdk: offline mode enabled with cache, but no cache is configured")
	})
	t.Run("redis enabled without addresses", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Cache: CacheConfig{Redis: RedisConfig{Enabled: true}}}}
		require.ErrorContains(t, conf.Validate(), "redis: at least 1 server address required")
	})
	t.Run("redis invalid tls config", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key", Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Key: "key"}}}}}}}
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")

		conf = Config{SDK: SDKConfig{Key: "Key", Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Cert: "cert"}}}}}}}
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")
	})
	t.Run("webhook signature invalid validity time", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key"}, Http: HttpConfig{Webhook: WebhookConfig{Enabled: true, SigningKey: "key", SignatureValidFor: 2}}}
		require.ErrorContains(t, conf.Validate(), "webhook: signature validity check must be greater than 5 seconds")
	})
	t.Run("webhook invalid auth", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key"}, Http: HttpConfig{Webhook: WebhookConfig{Enabled: true, Auth: AuthConfig{User: "user"}}}}
		require.ErrorContains(t, conf.Validate(), "webhook: both basic auth user and password required")

		conf = Config{SDK: SDKConfig{Key: "Key"}, Http: HttpConfig{Webhook: WebhookConfig{Enabled: true, Auth: AuthConfig{Password: "pass"}}}}
		require.ErrorContains(t, conf.Validate(), "webhook: both basic auth user and password required")
	})
	t.Run("http invalid tls config", func(t *testing.T) {
		conf := Config{SDK: SDKConfig{Key: "Key"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Key: "key"}}}}
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")

		conf = Config{SDK: SDKConfig{Key: "Key"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Cert: "cert"}}}}
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")
	})
}
