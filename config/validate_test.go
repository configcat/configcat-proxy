package config

import (
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("no envs", func(t *testing.T) {
		conf := Config{}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk: at least 1 SDK must be configured")
	})
	t.Run("no envs with auto key missing", func(t *testing.T) {
		conf := Config{AutoSDK: AutoSDKConfig{Key: "", Secret: "secret"}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk: at least 1 SDK must be configured")
	})
	t.Run("no envs with auto secret missing", func(t *testing.T) {
		conf := Config{AutoSDK: AutoSDKConfig{Key: "key", Secret: ""}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk: at least 1 SDK must be configured")
	})
	t.Run("invalid webhook valid for value", func(t *testing.T) {
		conf := Config{AutoSDK: AutoSDKConfig{Key: "key", Secret: "secret", WebhookSigningKey: "key", WebhookSignatureValidFor: -1}}
		conf.setDefaults()
		conf.fixupDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk: webhook signature validity check must be greater than 5 seconds")
	})
	t.Run("no envs with auto ok", func(t *testing.T) {
		conf := Config{AutoSDK: AutoSDKConfig{Key: "key", Secret: "secret", PollInterval: 60}}
		conf.setDefaults()
		require.NoError(t, conf.Validate())
	})
	t.Run("no envs with auto ok too low poll interval", func(t *testing.T) {
		conf := Config{AutoSDK: AutoSDKConfig{Key: "key", Secret: "secret", PollInterval: 1}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk: auto configuration poll interval cannot be less than 5 seconds")
	})
	t.Run("missing sdk key", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: SDK key is required")
	})
	t.Run("invalid data governance", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", DataGovernance: "inv"}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: invalid data governance value, it must be 'global' or 'eu'")
	})
	t.Run("offline invalid file path", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true, Local: LocalConfig{FilePath: "nonexisting"}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: couldn't find the local file")
	})
	t.Run("offline file polling invalid poll interval", func(t *testing.T) {
		testutils.UseTempFile("", func(path string) {
			conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true, Local: LocalConfig{FilePath: path, Polling: true, PollInterval: 0}}}}}
			conf.setDefaults()
			require.ErrorContains(t, conf.Validate(), "sdk-env1: local file poll interval must be greater than 1 seconds")
		})
	})
	t.Run("offline enabled without file and cache", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: offline mode requires either a configured cache or a local file")
	})
	t.Run("offline both local file and cache", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true, UseCache: true, Local: LocalConfig{FilePath: "file"}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: can't use both local file and cache for offline mode")
	})
	t.Run("offline cache without redis", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true, UseCache: true}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: offline mode enabled with cache, but no cache is configured")
	})
	t.Run("offline cache invalid poll interval", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", Offline: OfflineConfig{Enabled: true, UseCache: true, CachePollInterval: 0}}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: cache poll interval must be greater than 1 seconds")
	})
	t.Run("global offline cache invalid poll interval", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}}}, GlobalOfflineConfig: GlobalOfflineConfig{Enabled: true, CachePollInterval: -1}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "offline: cache poll interval must be greater than 1 seconds")
	})
	t.Run("global offline cache without cache", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, GlobalOfflineConfig: GlobalOfflineConfig{Enabled: true}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "offline: global offline mode enabled, but no cache is configured")
	})
	t.Run("mongo enabled without uri", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{MongoDb: MongoDbConfig{Enabled: true}}, Grpc: GrpcConfig{Port: 100}, Diag: DiagConfig{Port: 90}, Http: HttpConfig{Port: 80}}
		require.ErrorContains(t, conf.Validate(), "mongodb: invalid connection uri")
	})
	t.Run("mongodb invalid tls config", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{MongoDb: MongoDbConfig{Enabled: true, Url: "uri", Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Key: "key"}}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")

		conf = Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Cert: "cert"}}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")
	})
	t.Run("redis enabled without addresses", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true}}, Grpc: GrpcConfig{Port: 100}, Diag: DiagConfig{Port: 90}, Http: HttpConfig{Port: 80}}
		require.ErrorContains(t, conf.Validate(), "redis: at least 1 server address required")
	})
	t.Run("redis invalid tls config", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Key: "key"}}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")

		conf = Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Cache: CacheConfig{Redis: RedisConfig{Enabled: true, Addresses: []string{"localhost"}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Cert: "cert"}}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")
	})
	t.Run("webhook signature invalid validity time", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key", WebhookSigningKey: "key", WebhookSignatureValidFor: 2}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "sdk-env1: webhook signature validity check must be greater than 5 seconds")
	})
	t.Run("webhook invalid auth", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Http: HttpConfig{Webhook: WebhookConfig{Enabled: true, Auth: AuthConfig{User: "user"}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "webhook: both basic auth user and password required")

		conf = Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Http: HttpConfig{Webhook: WebhookConfig{Enabled: true, Auth: AuthConfig{Password: "pass"}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "webhook: both basic auth user and password required")
	})
	t.Run("http invalid tls config", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Key: "key"}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")

		conf = Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Tls: TlsConfig{Enabled: true, Certificates: []CertConfig{{Cert: "cert"}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "tls: both TLS cert and key file required")
	})
	t.Run("invalid ports", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Http: HttpConfig{Port: -1}}
			require.ErrorContains(t, conf.Validate(), "http: invalid port -1")
		})
		t.Run("metrics", func(t *testing.T) {
			conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Diag: DiagConfig{Port: -1}, Http: HttpConfig{Port: 80}}
			require.ErrorContains(t, conf.Validate(), "diag: invalid port -1")
		})
		t.Run("grpc", func(t *testing.T) {
			conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Grpc: GrpcConfig{Port: -1}, Diag: DiagConfig{Port: 90}, Http: HttpConfig{Port: 80}}
			require.ErrorContains(t, conf.Validate(), "grpc: invalid port -1")
		})
	})
	t.Run("cors regex invalid", func(t *testing.T) {
		conf := Config{SDKs: map[string]*SDKConfig{"env1": {Key: "Key"}}, Http: HttpConfig{Api: ApiConfig{Enabled: true, CORS: CORSConfig{Enabled: true, AllowedOriginsRegex: OriginRegexConfig{Patterns: []string{"*"}}}}}}
		conf.setDefaults()
		require.ErrorContains(t, conf.Validate(), "cors: the 'if no watch' field is required when allowed origins regex is set")
	})
}
