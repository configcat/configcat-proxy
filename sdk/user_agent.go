package sdk

import (
	"net/http"
)

const (
	configCatUserAgentHeader = "X-ConfigCat-UserAgent"
)

var proxyVersion = "0.0.0"

type userAgentInterceptor struct {
	http.RoundTripper
	userAgent string
}

func OverrideUserAgent(transport http.RoundTripper) http.RoundTripper {
	return &userAgentInterceptor{RoundTripper: transport, userAgent: "ConfigCat-Proxy/" + proxyVersion}
}

func (i *userAgentInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Header.Get(configCatUserAgentHeader) != "" {
		r.Header.Set(configCatUserAgentHeader, i.userAgent)
	}
	r.Header.Set("User-Agent", i.userAgent)
	return i.RoundTripper.RoundTrip(r)
}
