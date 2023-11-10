package mware

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"net/http"
	"slices"
	"strings"
)

var defaultAllowedHeaders = []string{
	"Cache-Control",
	"Content-Type",
	"Content-Length",
	"Accept-Encoding",
	"If-None-Match",
}

var defaultExposedHeaders = []string{
	"Content-Length",
	"ETag",
	"Date",
	"Content-Encoding",
}

var defaultAllowedOrigin = "*"

func CORS(allowedMethods []string, allowedOrigins []string, headers []string, authHeaders []string, originRegexConfig *config.OriginRegexConfig, next http.HandlerFunc) http.HandlerFunc {
	var exposedHeaders = defaultExposedHeaders
	if len(headers) > 0 {
		exposedHeaders = append(exposedHeaders, headers...)
		exposedHeaders = utils.DedupStringSlice(exposedHeaders)
	}

	var allowedHeaders = defaultAllowedHeaders
	if len(authHeaders) > 0 {
		allowedHeaders = append(allowedHeaders, authHeaders...)
		allowedHeaders = utils.DedupStringSlice(allowedHeaders)
	}

	exposedHeadersString := strings.Join(exposedHeaders, ",")
	allowedHeadersString := strings.Join(allowedHeaders, ",")

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if r.Method == http.MethodOptions {
			setOptionsCORSHeaders(w, origin, allowedOrigins, originRegexConfig, allowedMethods, exposedHeadersString, allowedHeadersString)
		} else {
			setDefaultCORSHeaders(w, origin, allowedOrigins, exposedHeadersString, originRegexConfig)
		}
		next(w, r)
	}
}

func setDefaultCORSHeaders(w http.ResponseWriter,
	requestOrigin string,
	allowedOrigins []string,
	exposedHeaders string,
	originRegexConfig *config.OriginRegexConfig) {
	w.Header().Set("Access-Control-Allow-Origin", determineOrigin(requestOrigin, allowedOrigins, originRegexConfig))
	w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
}

func setOptionsCORSHeaders(w http.ResponseWriter,
	requestOrigin string,
	allowedOrigins []string,
	originRegexConfig *config.OriginRegexConfig,
	allowedMethods []string,
	exposeHeaders string,
	allowedHeaders string) {
	setDefaultCORSHeaders(w, requestOrigin, allowedOrigins, exposeHeaders, originRegexConfig)
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
	if allowedMethods != nil && len(allowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	}
}

func determineOrigin(requestOrigin string, allowedOrigins []string, originRegexConfig *config.OriginRegexConfig) string {
	defaultResult := defaultAllowedOrigin

	hasAllowedOriginsSet := allowedOrigins != nil && len(allowedOrigins) > 0
	hasOriginRegexSet := originRegexConfig != nil && originRegexConfig.Regexes != nil && len(originRegexConfig.Regexes) > 0

	if hasAllowedOriginsSet {
		if slices.Contains(allowedOrigins, requestOrigin) {
			return requestOrigin
		} else {
			defaultResult = allowedOrigins[0]
		}
	}
	if hasOriginRegexSet {
		for _, regex := range originRegexConfig.Regexes {
			if regex.MatchString(requestOrigin) {
				return requestOrigin
			}
		}
		defaultResult = originRegexConfig.IfNoMatch
	}
	if !hasAllowedOriginsSet && !hasOriginRegexSet && requestOrigin != "" {
		return requestOrigin
	}
	return defaultResult
}
