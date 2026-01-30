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
	"github.com/testcontainers/testcontainers-go/modules/valkey"
)

type valkeyTestSuite struct {
	suite.Suite

	db     *valkey.ValkeyContainer
	dbPort string
}

func (s *valkeyTestSuite) SetupSuite() {
	valkeyContainer, err := valkey.Run(s.T().Context(), "valkey/valkey")
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}
	s.db = valkeyContainer
	p, _ := nat.NewPort("tcp", "6379")
	dbPort, _ := s.db.MappedPort(s.T().Context(), p)
	s.dbPort = dbPort.Port()
}

func (s *valkeyTestSuite) TearDownSuite() {
	if err := testcontainers.TerminateContainer(s.db); err != nil {
		panic("failed to terminate container: " + err.Error() + "")
	}
}

func TestValkeySuite(t *testing.T) {
	suite.Run(t, new(valkeyTestSuite))
}

func (s *valkeyTestSuite) TestValkeyStorage() {
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

func (s *valkeyTestSuite) TestValkeyStorage_Unavailable() {
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

func TestValkeyStorage_TLS(t *testing.T) {
	ctx := t.Context()

	valkeyContainer, err := redis.Run(ctx, "redis", redis.WithTLS())
	defer func() {
		if err := testcontainers.TerminateContainer(valkeyContainer); err != nil {
			panic("failed to terminate container: " + err.Error() + "")
		}
	}()
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}

	p, _ := nat.NewPort("tcp", "6379")
	dbPort, _ := valkeyContainer.MappedPort(t.Context(), p)

	tls := valkeyContainer.TLSConfig()

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
