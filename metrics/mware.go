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

func (rec *requestInterceptor) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
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
