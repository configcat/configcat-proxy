package telemetry

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	codes2 "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	otlptpb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tpb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestOtlpTracesExporterGrpc(t *testing.T) {
	collector, err := newInMemoryTraceGrpcCollector()
	assert.NoError(t, err)
	defer collector.Shutdown()

	handler := newTraceHandler(t.Context(), buildResource("0.1.0"),
		&config.TraceConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "grpc", Endpoint: collector.Addr()}}, log.NewNullLogger())
	assert.NotNil(t, handler)
	defer handler.Shutdown()

	_, s := handler.provider.Tracer("t").Start(t.Context(), "span", trace.WithSpanKind(trace.SpanKindClient))
	s.SetStatus(codes2.Ok, "OK")
	s.End()
	_ = handler.provider.ForceFlush(t.Context())

	assert.True(t, collector.hasTrace("span"))
}

func TestOtlpTracesExporterHttp(t *testing.T) {
	collector := newInMemoryTraceHttpCollector()
	defer collector.Shutdown()

	handler := newTraceHandler(t.Context(), buildResource("0.1.0"),
		&config.TraceConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "http", Endpoint: collector.Addr()}}, log.NewNullLogger())
	assert.NotNil(t, handler)
	defer handler.Shutdown()

	_, s := handler.provider.Tracer("t").Start(t.Context(), "span", trace.WithSpanKind(trace.SpanKindClient))
	s.SetStatus(codes2.Ok, "OK")
	s.End()
	_ = handler.provider.ForceFlush(t.Context())

	assert.True(t, hasTrace(collector, "span"))
}

type grpcCollector struct {
	listener net.Listener
	srv      *grpc.Server
	mu       sync.RWMutex
}

type inMemoryTraceGrpcCollector struct {
	otlptpb.UnimplementedTraceServiceServer
	*grpcCollector

	traces []*tpb.ResourceSpans
}

func newGrpcCollector() (*grpcCollector, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer()
	return &grpcCollector{
		listener: listener,
		srv:      srv,
	}, nil
}

func newInMemoryTraceGrpcCollector() (*inMemoryTraceGrpcCollector, error) {
	gc, err := newGrpcCollector()
	if err != nil {
		return nil, err
	}
	c := &inMemoryTraceGrpcCollector{
		grpcCollector: gc,
		traces:        make([]*tpb.ResourceSpans, 0),
	}
	otlptpb.RegisterTraceServiceServer(c.srv, c)
	go func() { _ = c.srv.Serve(c.listener) }()

	return c, nil
}

func (c *grpcCollector) Addr() string {
	return c.listener.Addr().String()
}

func (c *grpcCollector) Shutdown() {
	c.srv.Stop()
}

func (c *inMemoryTraceGrpcCollector) hasTrace(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, t := range c.traces {
		for _, span := range t.ScopeSpans {
			for _, s := range span.Spans {
				if s.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func (c *inMemoryTraceGrpcCollector) Export(_ context.Context, req *otlptpb.ExportTraceServiceRequest) (*otlptpb.ExportTraceServiceResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.traces = append(c.traces, req.ResourceSpans...)

	return &otlptpb.ExportTraceServiceResponse{}, nil
}

func newInMemoryTraceHttpCollector() *inMemoryHttpCollector[*otlptpb.ExportTraceServiceRequest] {
	c := &inMemoryHttpCollector[*otlptpb.ExportTraceServiceRequest]{
		records: make([]*otlptpb.ExportTraceServiceRequest, 0),
	}
	c.srv = httptest.NewServer(c)
	return c
}

type inMemoryHttpCollector[T proto.Message] struct {
	srv     *httptest.Server
	mu      sync.RWMutex
	records []T
}

func hasTrace(c *inMemoryHttpCollector[*otlptpb.ExportTraceServiceRequest], name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, r := range c.records {
		for _, t := range r.ResourceSpans {
			for _, span := range t.ScopeSpans {
				for _, s := range span.Spans {
					if s.Name == name {
						return true
					}
				}
			}
		}
	}
	return false
}

func (c *inMemoryHttpCollector[T]) Addr() string {
	return c.srv.Listener.Addr().String()
}

func (c *inMemoryHttpCollector[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var reader io.ReadCloser
	var err error
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, err.Error())
			return
		}
	default:
		reader = r.Body
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, err.Error())
		return
	}

	var req T
	msgType := reflect.TypeOf(req).Elem()
	req = reflect.New(msgType).Interface().(T)
	err = proto.Unmarshal(body, req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, err.Error())
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, req)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (c *inMemoryHttpCollector[msg]) Shutdown() {
	c.srv.Close()
}
