package mware

import (
	"github.com/configcat/configcat-proxy/log"
	"net/http"
	"time"
)

type requestInterceptor struct {
	http.ResponseWriter

	statusCode     int
	responseLength uint64
}

func (r *requestInterceptor) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *requestInterceptor) Write(data []byte) (int, error) {
	r.responseLength += uint64(len(data))
	return r.ResponseWriter.Write(data)
}

func (r *requestInterceptor) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func DebugLog(log log.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := requestInterceptor{w, http.StatusOK, 0}

		log.Debugf("request starting %s %s %s", r.Proto, r.Method, r.URL)
		next(&interceptor, r)
		duration := time.Since(start)
		log.Debugf("request finished %s %s %s [status: %d] [duration: %dms] [response: %dB]",
			r.Proto, r.Method, r.URL, interceptor.statusCode, duration.Milliseconds(), interceptor.responseLength)
	}
}
