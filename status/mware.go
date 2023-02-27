package status

import (
	"fmt"
	"net/http"
)

type clientInterceptor struct {
	http.RoundTripper
	reporter Reporter
}

func Intercept(reporter Reporter, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{reporter: reporter, RoundTripper: transport}
}

func (i *clientInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := i.RoundTripper.RoundTrip(r)
	if err != nil {
		i.reporter.ReportError(SDK, err)
	} else {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			i.reporter.ReportOk(SDK, "config fetched")
		} else if resp.StatusCode == http.StatusNotModified {
			i.reporter.ReportOk(SDK, "config not modified")
		} else {
			i.reporter.ReportError(SDK, fmt.Errorf("unexpected response received: %s", resp.Status))
		}
	}
	return resp, err
}
