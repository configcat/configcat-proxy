package mware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

type gzipWriter struct {
	http.ResponseWriter

	writer      *gzip.Writer
	headersDone bool
}

func GZip(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			writer := pool.Get().(*gzip.Writer)
			writer.Reset(w)
			gz := &gzipWriter{writer: writer, ResponseWriter: w}
			defer func() {
				_ = gz.Close()
			}()
			next(gz, r)
		} else {
			next(w, r)
		}
	}
}

func (w *gzipWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
	w.headersDone = true
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	if !w.headersDone {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(b)
}

func (w *gzipWriter) Flush() {
	_ = w.writer.Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipWriter) Close() error {
	defer pool.Put(w.writer)
	return w.writer.Close()
}
