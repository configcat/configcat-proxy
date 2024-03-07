package metrics

import (
	"net/http"
	"strconv"
	"time"
)

type requestInterceptor struct {
	http.ResponseWriter

	statusCode int
}

func (r *requestInterceptor) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func Measure(metricsHandler Reporter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := requestInterceptor{w, http.StatusOK}

		next(&interceptor, r)

		duration := time.Since(start)
		metricsHandler.(*reporter).responseTime.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(interceptor.statusCode)).Observe(duration.Seconds())
	}
}

type clientInterceptor struct {
	http.RoundTripper

	metricsHandler Reporter
	sdkId          string
}

func InterceptSdk(sdkId string, metricsHandler Reporter, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{metricsHandler: metricsHandler, RoundTripper: transport, sdkId: sdkId}
}

func (i *clientInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := i.RoundTripper.RoundTrip(r)
	duration := time.Since(start)
	var stat string
	if err != nil {
		stat = err.Error()
	} else {
		stat = resp.Status
	}
	i.metricsHandler.(*reporter).sdkResponseTime.WithLabelValues(i.sdkId, r.URL.String(), stat).Observe(duration.Seconds())
	return resp, err
}
