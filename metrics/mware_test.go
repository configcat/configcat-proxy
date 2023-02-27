package metrics

import (
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeasure(t *testing.T) {
	handler := NewHandler().(*handler)
	h := Measure(handler, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	expected := `
# HELP configcat_http_request_duration_seconds Histogram of HTTP response time in seconds.
# TYPE configcat_http_request_duration_seconds histogram
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.005"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.01"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.025"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.05"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.1"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.25"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="0.5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="1"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="2.5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="10"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="200",le="+Inf"} 1
configcat_http_request_duration_seconds_sum{method="GET",route="/",status="200"} 0
configcat_http_request_duration_seconds_count{method="GET",route="/",status="200"} 1

`

	assert.NoError(t, testutil.CollectAndCompare(handler.responseTime, strings.NewReader(expected)))
}

func TestMeasure_Non_Success(t *testing.T) {
	handler := NewHandler().(*handler)
	h := Measure(handler, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	expected := `
# HELP configcat_http_request_duration_seconds Histogram of HTTP response time in seconds.
# TYPE configcat_http_request_duration_seconds histogram
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.005"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.01"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.025"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.05"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.1"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.25"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="0.5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="1"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="2.5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="5"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="10"} 1
configcat_http_request_duration_seconds_bucket{method="GET",route="/",status="400",le="+Inf"} 1
configcat_http_request_duration_seconds_sum{method="GET",route="/",status="400"} 0
configcat_http_request_duration_seconds_count{method="GET",route="/",status="400"} 1

`

	assert.NoError(t, testutil.CollectAndCompare(handler.responseTime, strings.NewReader(expected)))
}

func TestIntercept(t *testing.T) {
	handler := NewHandler().(*handler)
	h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	client.Transport = InterceptSdk(handler, http.DefaultTransport)
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.sdkResponseTime))
}
