package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestMeasure(t *testing.T) {
	handler := NewReporter().(*reporter)
	h := Measure(handler, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.httpResponseTime))

	mSrv := httptest.NewServer(handler.HttpHandler())
	client = http.Client{}
	req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Contains(t, string(body), "configcat_http_request_duration_seconds_bucket{method=\"GET\",route=\"/\",status=\"200\",le=\"0.005\"} 1")
}

func TestMeasure_Non_Success(t *testing.T) {
	handler := NewReporter().(*reporter)
	h := Measure(handler, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.httpResponseTime))

	mSrv := httptest.NewServer(handler.HttpHandler())
	client = http.Client{}
	req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Contains(t, string(body), "configcat_http_request_duration_seconds_bucket{method=\"GET\",route=\"/\",status=\"400\",le=\"0.005\"} 1")
}

func TestIntercept(t *testing.T) {
	handler := NewReporter().(*reporter)
	h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	client.Transport = InterceptSdk("test", handler, http.DefaultTransport)
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.sdkResponseTime))

	mSrv := httptest.NewServer(handler.HttpHandler())
	client = http.Client{}
	req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Contains(t, string(body), "configcat_sdk_http_request_duration_seconds_bucket{route=\""+srv.URL+"\",sdk=\"test\",status=\"200 OK\",le=\"0.005\"} 1")
}

func TestProfileIntercept(t *testing.T) {
	handler := NewReporter().(*reporter)
	h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	client := http.Client{}
	client.Transport = InterceptProxyProfile("test", handler, http.DefaultTransport)
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	_, _ = client.Do(req)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.profileResponseTime))

	mSrv := httptest.NewServer(handler.HttpHandler())
	client = http.Client{}
	req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Contains(t, string(body), "configcat_profile_http_request_duration_seconds_bucket{key=\"test\",route=\""+srv.URL+"\",status=\"200 OK\",le=\"0.005\"} 1")
}

func TestUnaryInterceptor(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (i interface{}, e error) {
		return nil, nil
	}

	rep := NewReporter().(*reporter)
	i := GrpcUnaryInterceptor(rep)
	_, err := i(context.Background(), "test-req", &grpc.UnaryServerInfo{FullMethod: "test-method"}, handler)

	assert.NoError(t, err)
	assert.Equal(t, 1, testutil.CollectAndCount(rep.grpcResponseTime))

	mSrv := httptest.NewServer(rep.HttpHandler())
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Contains(t, string(body), "configcat_grpc_rpc_duration_seconds_bucket{code=\"OK\",method=\"test-method\",le=\"0.005\"} 1")
}
