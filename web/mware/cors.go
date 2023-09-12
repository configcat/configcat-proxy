package mware

import (
	"net/http"
	"strings"
)

var defaultAllowedHeaders = strings.Join([]string{
	"Cache-Control",
	"Content-Type",
	"Content-Length",
	"Accept-Encoding",
	"If-None-Match",
}, ",")

var defaultExposedHeaders = strings.Join([]string{
	"Content-Length",
	"ETag",
	"Date",
	"Content-Encoding",
}, ",")

var defaultAllowedOrigin = "*"

func CORS(allowedMethods []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if r.Method == http.MethodOptions {
			setOptionsCORSHeaders(w, origin, allowedMethods)
		} else {
			setDefaultCORSHeaders(w, origin)
		}
		next(w, r)
	}
}

func setDefaultCORSHeaders(w http.ResponseWriter, origin string) {
	if origin == "" {
		origin = defaultAllowedOrigin
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Expose-Headers", defaultExposedHeaders)
}

func setOptionsCORSHeaders(w http.ResponseWriter, origin string, allowedMethods []string) {
	setDefaultCORSHeaders(w, origin)
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Set("Access-Control-Allow-Headers", defaultAllowedHeaders)
	if allowedMethods != nil && len(allowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	}
}
