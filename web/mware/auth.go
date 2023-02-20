package mware

import (
	"crypto/sha256"
	"crypto/subtle"
	"github.com/configcat/configcat-proxy/log"
	"net/http"
)

func BasicAuth(user string, pass string, logger log.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			logger.Debugf("basic auth is configured but it's missing from the request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Hash to prevent timing attack
		userHash := sha256.Sum256([]byte(username))
		passHash := sha256.Sum256([]byte(password))
		expUserHash := sha256.Sum256([]byte(user))
		expPassHash := sha256.Sum256([]byte(pass))
		userMatch := subtle.ConstantTimeCompare(userHash[:], expUserHash[:]) == 1
		passMatch := subtle.ConstantTimeCompare(passHash[:], expPassHash[:]) == 1
		if !userMatch || !passMatch {
			logger.Debugf("basic auth credential validation failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func HeaderAuth(authHeaders map[string]string, logger log.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range authHeaders {
			h := r.Header.Get(k)
			if h != v {
				logger.Debugf("auth header (%s) validation failed", k)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}
