package status

import (
	"fmt"
	"net/http"
)

type clientInterceptor struct {
	http.RoundTripper

	reporter Reporter
	envId    string
}

func InterceptSdk(envId string, reporter Reporter, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{reporter: reporter, RoundTripper: transport, envId: envId}
}

func (i *clientInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := i.RoundTripper.RoundTrip(r)
	if err != nil {
		i.reporter.ReportError(i.envId, err)
	} else {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			i.reporter.ReportOk(i.envId, "config fetched")
		} else if resp.StatusCode == http.StatusNotModified {
			i.reporter.ReportOk(i.envId, "config not modified")
		} else {
			i.reporter.ReportError(i.envId, fmt.Errorf("unexpected response received: %s", resp.Status))
		}
	}
	return resp, err
}
