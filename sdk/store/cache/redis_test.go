package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisStorage(t *testing.T) {
	s := miniredis.RunT(t)
	store, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(context.Background(), "key", cacheEntry)
	assert.NoError(t, err)
	s.CheckGet(t, "key", string(cacheEntry))
	res, err := srv.Get(context.Background(), "key")
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(j))
}

func TestRedisStorage_Unavailable(t *testing.T) {
	store, err := newRedis(&config.RedisConfig{Addresses: []string{"nonexisting"}}, log.NewNullLogger())
	assert.NoError(t, err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(context.Background(), "", cacheEntry)
	assert.Error(t, err)
	_, err = srv.Get(context.Background(), "")
	assert.Error(t, err)
}

func TestRedisStorage_TLS(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		utils.UseTempFile(`
-----BEGIN CERTIFICATE-----
MIICrzCCAZcCFDnpdKF+Pg1smjtIXrNdIgxGYEJfMA0GCSqGSIb3DQEBCwUAMBQx
EjAQBgNVBAMMCWxvY2FsaG9zdDAeFw0yMzAzMDEyMTA2NThaFw0yNDAyMjkyMTA2
NThaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAOiTDTjfAPvJLDZ2mwNvu0pohSHPRzzfZRc16iVI6+ESl0Dwjdjl
yERFO/ts1GQnhE2ggykvoxH4zUy1OCnjTJ+Mm1ryjy4G5ZIILIF9MfFcyma5/5Xd
oOTcDr3ZDTAwFaabKYKisoVMHAJCphencgoyOToW5/HRHMKOEpTJOQWSyNduXYfY
nsWb3hx7WD9NajliW7/Jjbf7UnDtKY2VM2GZWT3ygIH/7SlBqyuXJNqyZXbqfbrP
6mdZQ5wvYsnSUU4kNMtZg/ns+0H5R7PFmRhIRM0nZvJZTO9oHREdm+e2nnZwHyJF
Z26LxE7Qr1bn8+PQSydyQIqeUdaSX2LuXqECAwEAATANBgkqhkiG9w0BAQsFAAOC
AQEAjRoOTe4W4OQ6YOo5kx5sMAozh0Rg6eifS0s8GuxKwfuBop8FEnM3wAfF6x3J
fsik9MmoM4L11HWjttb46UFq/rP3GsA3DLX8i1yBOES+iyCELd5Ss9q1jfr/Jqo3
cAanE4yl3NNEZoDmMdSj2U11BneKSzHDR+l2hDF9wBifWGI9DQ1ItfA5I6MwnL+0
J03vcwPSwme4bKC/avAT2oDD7jLGLA+kuhMqHvVq7nXRzs46xyFPBBv7fBxXjPPG
c89d0ISafKtZ9kIKaRrzu2HX+b0fzKr0vtHYDLtC1U5oU7GPB12eupERkmWYlhrw
hDL3X7kt3jEZFkzGV1XL1IJx/g==
-----END CERTIFICATE-----`, func(cert string) {
			utils.UseTempFile(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDokw043wD7ySw2
dpsDb7tKaIUhz0c832UXNeolSOvhEpdA8I3Y5chERTv7bNRkJ4RNoIMpL6MR+M1M
tTgp40yfjJta8o8uBuWSCCyBfTHxXMpmuf+V3aDk3A692Q0wMBWmmymCorKFTBwC
QqYXp3IKMjk6Fufx0RzCjhKUyTkFksjXbl2H2J7Fm94ce1g/TWo5Ylu/yY23+1Jw
7SmNlTNhmVk98oCB/+0pQasrlyTasmV26n26z+pnWUOcL2LJ0lFOJDTLWYP57PtB
+UezxZkYSETNJ2byWUzvaB0RHZvntp52cB8iRWdui8RO0K9W5/Pj0EsnckCKnlHW
kl9i7l6hAgMBAAECggEBAOMWiqeIH5a6BGCdiJhfZZmu2qd7k8xdOIDkVN7ZB/B5
TZTMDUTGgLggfgPubKfqaeW+H7N8XxZyQEtw+wjzduKm0R6JjsJbW5cuQf6htr08
ZCjP3j5/69TrBb3bjGQL32gRQwPaRsOe4A5Y84JPLivEhFoy+YEFNLbHMF905yeH
IaSeqeK0GNm0a/MU68pa1ODIc8B2zqo+f6I9qekezlDR7Or487FqnlLtNf0yvnLD
sbshzj5rzLdLYgA/RNZ4CkuGddxEYjnDB1IG0NX8m9MrHlsi7jqxa7pHt5oDrRsW
ZxBez6Q70dE29sdl5lnce3qjxweB2NK3Q6Cr2eyizwECgYEA/L/WzgY1yDMWzaCr
SRThg9NWO1EYbvz4uxt7rElfZ+NYAaT08E35Ooo9IeBzp3VoFA1PcNQnKB5pgczO
Mu5W/td5zpx1dzguBZAl4IpKkml08i06R7FxxTqtRM/P7Pna+RagtqAo3JZww3bd
ofIPH2OrobqlcFhOsLqKp5ocDNECgYEA65DJsImeBfW1aZ5ABgPr7NErSv2fKj1r
eGsgC5Za1ZiaG5LWkCpuezsvf6ma4EN3CMl5Fo617qaY6mnL2HlfVtFhHYSeLpna
9ZgqZ1zj2HkqiXOPEkb3d3cC61rXiMK97NpshrpzFx+uMCH8MMu9/CVJEHNKGgAq
6zZQ4LhjaNECgYEA3W4UeprmM2bO64d/iJ9Kk3traLw7c8EdCI+jYeVGOHXsfERQ
ctddKfRCapOBv4wUiry+hFLZm0RJmvYbEHPOs6WDiYd5QeFuMGGBTZ7ahjrtwd3t
2TGUQv6NHmQR/cNIHEG+u0DFi7whPp28vkybAx0HGMG0fyBekGZdY0iYmoECgYEA
3mVOlVYHk9ba1AEsrsErDuSXe/AgQa/E8+YnVek4jqnI7LlfyrHUppFFEcDdUFdB
XVFg+ZP4XXx5p+4EHrbP9NYuWsDm2lY1K2Livb0r+ybBqw0niPjpD6eTYQHdtOcu
ihvZFAWZPL6TJCwhvSvNjOziox5FWnDIFFKuXsqWR9ECgYAfiG1izToF+GX3yUPq
CU+ceTbM2uy3hVnQLvCnraN7hkF02Fa9ZwP6nmnsvhfdaIUP5WLm3A+qMWu/PL0i
F/dUCUF6M/DyihQUnOl+MD9Sg89ZHiftqXSY8jGR14uH4woStyUFHiFbtajmnqV7
MK4Li/LGWcksyoF+hbPNXMFCIA==
-----END PRIVATE KEY-----
`, func(key string) {
				s := miniredis.RunT(t)
				store, err := newRedis(&config.RedisConfig{
					Addresses: []string{s.Addr()},
					Tls: config.TlsConfig{
						Enabled:    true,
						MinVersion: 1.1,
						Certificates: []config.CertConfig{
							{Key: key, Cert: cert},
						},
					},
				}, log.NewNullLogger())
				assert.NoError(t, err)
				assert.NotNil(t, store)
			})
		})
	})
	t.Run("invalid", func(t *testing.T) {
		s := miniredis.RunT(t)
		store, err := newRedis(&config.RedisConfig{
			Addresses: []string{s.Addr()},
			Tls: config.TlsConfig{
				Enabled:    true,
				MinVersion: 1.1,
				Certificates: []config.CertConfig{
					{Key: "nonexisting", Cert: "nonexisting"},
				},
			},
		}, log.NewNullLogger())
		assert.ErrorContains(t, err, "failed to load certificate and key files")
		assert.Nil(t, store)
	})
}
