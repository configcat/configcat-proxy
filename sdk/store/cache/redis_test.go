package cache

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

type redisTestSuite struct {
	suite.Suite

	db     *redis.RedisContainer
	dbPort string
}

func (s *redisTestSuite) SetupSuite() {
	redisContainer, err := redis.Run(s.T().Context(), "redis")
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}
	s.db = redisContainer
	p, _ := nat.NewPort("tcp", "6379")
	dbPort, _ := s.db.MappedPort(s.T().Context(), p)
	s.dbPort = dbPort.Port()
}

func (s *redisTestSuite) TearDownSuite() {
	if err := testcontainers.TerminateContainer(s.db); err != nil {
		panic("failed to terminate container: " + err.Error() + "")
	}
}

func TestRedisSuite(t *testing.T) {
	suite.Run(t, new(redisTestSuite))
}

func (s *redisTestSuite) TestRedisStorage() {
	store, err := newRedis(&config.RedisConfig{Addresses: []string{"localhost:" + s.dbPort}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(s.T().Context(), "key", cacheEntry)
	assert.NoError(s.T(), err)
	res, err := srv.Get(s.T().Context(), "key")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), cacheEntry, res)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), `test`, string(j))
}

func (s *redisTestSuite) TestRedisStorage_Unavailable() {
	store, err := newRedis(&config.RedisConfig{Addresses: []string{"nonexisting"}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(s.T().Context(), "", cacheEntry)
	assert.Error(s.T(), err)
	_, err = srv.Get(s.T().Context(), "")
	assert.Error(s.T(), err)
}

func TestRedisStorage_TLS(t *testing.T) {
	ctx := t.Context()

	redisContainer, err := redis.Run(ctx, "redis", redis.WithTLS())
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			panic("failed to terminate container: " + err.Error() + "")
		}
	}()
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}

	p, _ := nat.NewPort("tcp", "6379")
	dbPort, _ := redisContainer.MappedPort(t.Context(), p)

	tls := redisContainer.TLSConfig()

	cert := tls.Certificates[0]

	var pemCerts [][]byte
	for _, derBytes := range cert.Certificate {
		pemBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		}
		pemBytes := pem.EncodeToMemory(pemBlock)
		pemCerts = append(pemCerts, pemBytes)
	}

	pk, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	assert.NoError(t, err)
	k := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pk,
	})

	t.Run("valid", func(t *testing.T) {
		sep := []byte{'\n'}
		testutils.UseTempFile(string(bytes.Join(pemCerts, sep)), func(cert string) {
			testutils.UseTempFile(string(k), func(key string) {
				store, err := newRedis(&config.RedisConfig{
					Addresses: []string{"localhost:" + dbPort.Port()},
					Tls: config.TlsConfig{
						Enabled:    true,
						MinVersion: 1.1,
						Certificates: []config.CertConfig{
							{Key: key, Cert: cert},
						},
					},
				}, telemetry.NewEmptyReporter(), log.NewNullLogger())
				assert.NoError(t, err)
				assert.NotNil(t, store)
			})
		})
	})
	t.Run("invalid", func(t *testing.T) {
		store, err := newRedis(&config.RedisConfig{
			Addresses: []string{"localhost:" + dbPort.Port()},
			Tls: config.TlsConfig{
				Enabled:    true,
				MinVersion: 1.1,
				Certificates: []config.CertConfig{
					{Key: "nonexisting", Cert: "nonexisting"},
				},
			},
		}, telemetry.NewEmptyReporter(), log.NewNullLogger())
		assert.ErrorContains(t, err, "failed to load certificate and key files")
		assert.Nil(t, store)
	})
}
