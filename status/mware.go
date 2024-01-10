package status

import (
	"fmt"
	"net/http"
)

type clientInterceptor struct {
	http.RoundTripper

	reporter Reporter
	sdkId    string
}

func InterceptSdk(sdkId string, reporter Reporter, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{reporter: reporter, RoundTripper: transport, sdkId: sdkId}
}

func (i *clientInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := i.RoundTripper.RoundTrip(r)
	if err != nil {
		i.reporter.ReportError(i.sdkId, "config fetch failed")
	} else {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			i.reporter.ReportOk(i.sdkId, "config fetched")
		} else if resp.StatusCode == http.StatusNotModified {
			i.reporter.ReportOk(i.sdkId, "config not modified")
		} else {
			i.reporter.ReportError(i.sdkId, fmt.Sprintf("unexpected response received: %s", resp.Status))
		}
	}
	return resp, err
}
