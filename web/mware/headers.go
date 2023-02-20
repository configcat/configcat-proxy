package mware

import "net/http"

func ExtraHeaders(headers map[string]string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		next(w, r)
	}
}
