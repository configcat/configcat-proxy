package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type httpRequestInterceptor struct {
	http.ResponseWriter

	statusCode int
}

func (r *httpRequestInterceptor) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func Measure(metricsReporter Reporter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := httpRequestInterceptor{w, http.StatusOK}

		next(&interceptor, r)

		duration := time.Since(start)
		metricsReporter.(*reporter).httpResponseTime.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(interceptor.statusCode)).Observe(duration.Seconds())
	}
}

func GrpcUnaryInterceptor(metricsReporter Reporter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		stat, ok := status.FromError(err)
		if !ok {
			stat = status.FromContextError(err)
		}
		duration := time.Since(start)
		metricsReporter.(*reporter).grpcResponseTime.WithLabelValues(info.FullMethod, stat.Code().String()).Observe(duration.Seconds())
		return resp, err
	}
}

type clientInterceptor struct {
	http.RoundTripper

	metricsHandler Reporter
	sdkId          string
}

func InterceptSdk(sdkId string, metricsHandler Reporter, transport http.RoundTripper) http.RoundTripper {
	return &clientInterceptor{metricsHandler: metricsHandler, RoundTripper: transport, sdkId: sdkId}
}

func (i *clientInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := i.RoundTripper.RoundTrip(r)
	duration := time.Since(start)
	var stat string
	if err != nil {
		stat = err.Error()
	} else {
		stat = resp.Status
	}
	i.metricsHandler.(*reporter).sdkResponseTime.WithLabelValues(i.sdkId, r.URL.String(), stat).Observe(duration.Seconds())
	return resp, err
}

type profileInterceptor struct {
	http.RoundTripper

	metricsHandler Reporter
	key            string
}

func InterceptProxyProfile(key string, metricsHandler Reporter, transport http.RoundTripper) http.RoundTripper {
	return &profileInterceptor{metricsHandler: metricsHandler, RoundTripper: transport, key: key}
}

func (p *profileInterceptor) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := p.RoundTripper.RoundTrip(r)
	duration := time.Since(start)
	var stat string
	if err != nil {
		stat = err.Error()
	} else {
		stat = resp.Status
	}
	p.metricsHandler.(*reporter).profileResponseTime.WithLabelValues(p.key, r.URL.String(), stat).Observe(duration.Seconds())
	return resp, err
}
