package sdk

import (
	"net/http"
)

var proxyVersion = "0.0.0"

type userAgentInterceptor struct {
	http.RoundTripper
}

func OverrideUserAgent(transport http.RoundTripper) http.RoundTripper {
	return &userAgentInterceptor{RoundTripper: transport}
}

func (i *userAgentInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-ConfigCat-UserAgent", "ConfigCat-Proxy/"+proxyVersion)
	return i.RoundTripper.RoundTrip(r)
}
