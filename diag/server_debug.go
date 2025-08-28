//go:build debug

package diag

import (
	"net/http"
	"net/http/pprof"
)

func setupDebugEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/{action}", pprof.Index)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}
