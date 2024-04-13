package sdk

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgentInterceptor_RoundTrip(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("X-ConfigCat-UserAgent"), "ConfigCat-Proxy/2.0.1")
			w.WriteHeader(http.StatusOK)
		}))
	defer ts.Close()
	client := &http.Client{
		Transport: OverrideUserAgent(http.DefaultTransport),
	}
	proxyVersion = "2.0.1"
	_, err := client.Get(ts.URL)
	assert.NoError(t, err)
}

func TestUserAgentInterceptor_RoundTrip_ExistingHeader(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("X-ConfigCat-UserAgent"), "ConfigCat-Proxy/2.0.1")
			w.WriteHeader(http.StatusOK)
		}))
	defer ts.Close()
	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	req.Header.Set("X-ConfigCat-UserAgent", "other")
	client := &http.Client{
		Transport: OverrideUserAgent(http.DefaultTransport),
	}
	proxyVersion = "2.0.1"
	_, err := client.Do(req)
	assert.NoError(t, err)
}
