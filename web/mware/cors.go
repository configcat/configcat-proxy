package mware

import (
	"net/http"
	"slices"
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

func CORS(allowedMethods []string, allowedOrigins []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if r.Method == http.MethodOptions {
			setOptionsCORSHeaders(w, origin, allowedOrigins, allowedMethods)
		} else {
			setDefaultCORSHeaders(w, origin, allowedOrigins)
		}
		next(w, r)
	}
}

func setDefaultCORSHeaders(w http.ResponseWriter, requestOrigin string, allowedOrigins []string) {
	w.Header().Set("Access-Control-Allow-Origin", determineOrigin(requestOrigin, allowedOrigins))
	w.Header().Set("Access-Control-Expose-Headers", defaultExposedHeaders)
}

func setOptionsCORSHeaders(w http.ResponseWriter, requestOrigin string, allowedOrigins []string, allowedMethods []string) {
	setDefaultCORSHeaders(w, requestOrigin, allowedOrigins)
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Set("Access-Control-Allow-Headers", defaultAllowedHeaders)
	if allowedMethods != nil && len(allowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	}
}

func determineOrigin(requestOrigin string, allowedOrigins []string) string {
	if allowedOrigins != nil && len(allowedOrigins) > 0 {
		if slices.Contains(allowedOrigins, requestOrigin) {
			return requestOrigin
		} else {
			return allowedOrigins[0]
		}
	}
	if requestOrigin != "" {
		return requestOrigin
	}
	return defaultAllowedOrigin
}
