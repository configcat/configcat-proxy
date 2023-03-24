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

func Measure(metricsHandler Handler, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := requestInterceptor{w, http.StatusOK}

		next(&interceptor, r)

		duration := time.Since(start)
		metricsHandler.(*handler).responseTime.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(interceptor.statusCode)).Observe(duration.Seconds())
	}
}

type clientInterceptor struct {
	http.RoundTripper

	metricsHandler Handler
	envId          string
}

func InterceptSdk(envId string, metricsHandler Handler, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{metricsHandler: metricsHandler, RoundTripper: transport, envId: envId}
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
	i.metricsHandler.(*handler).sdkResponseTime.WithLabelValues(i.envId, r.URL.String(), stat).Observe(duration.Seconds())
	return resp, err
}
