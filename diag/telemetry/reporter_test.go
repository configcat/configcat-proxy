package telemetry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	tcdynamodb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

func TestHandler_Metrics_Prometheus_Export(t *testing.T) {
	conf := config.DiagConfig{
		Port:    5052,
		Enabled: true,
		Metrics: config.MetricsConfig{Enabled: true, Prometheus: config.PrometheusExporterConfig{Enabled: true}},
	}

	handler := NewReporter(&conf, "0.1.0", log.NewNullLogger()).(*reporter)
	defer handler.Shutdown()

	mSrv := httptest.NewServer(handler.GetPrometheusHttpHandler())
	defer mSrv.Close()

	t.Run("http", func(t *testing.T) {
		h := handler.InstrumentHttp("t", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_server_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"200\",http_route=\"/\",network_protocol_name=\"http\",network_protocol_version=\"1.1\"")
	})
	t.Run("http bad", func(t *testing.T) {
		h := handler.InstrumentHttp("t", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_server_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"400\",http_route=\"/\",network_protocol_name=\"http\",network_protocol_version=\"1.1\"")
	})
	t.Run("http client", func(t *testing.T) {
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		client := http.Client{}
		client.Transport = handler.InstrumentHttpClient(http.DefaultTransport, NewKV("sdk", "test"))
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_client_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"200\",http_route=\"\",network_protocol_name=\"http\"")
	})
	t.Run("grpc", func(t *testing.T) {
		opts := handler.InstrumentGrpc([]grpc.ServerOption{})
		assert.Equal(t, 1, len(opts))
	})
	t.Run("conn", func(t *testing.T) {
		handler.RecordConnections(5, "sdk", "grpc", "flag")

		req, _ := http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_stream_connections{flag=\"flag\"")
	})
	t.Run("sent msgs", func(t *testing.T) {
		handler.AddSentMessageCount(5, "sdk", "grpc", "flag")

		req, _ := http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_stream_msg_sent_total{flag=\"flag\"")
	})
	t.Run("redis", func(t *testing.T) {
		r := miniredis.RunT(t)
		opts := &redis.UniversalOptions{Addrs: []string{r.Addr()}}
		rdb := redis.NewUniversalClient(opts)
		handler.InstrumentRedis(rdb)

		req, _ := http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_db_client_connections_waits{db_system=\"redis\"")
	})
}

func TestHandler_Metrics_Otlp_Export(t *testing.T) {
	collector, err := newInMemoryMetricGrpcCollector()
	assert.NoError(t, err)
	defer collector.Shutdown()
	conf := config.DiagConfig{
		Port:    5052,
		Enabled: true,
		Metrics: config.MetricsConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "grpc", Endpoint: collector.Addr()}},
	}

	handler := NewReporter(&conf, "0.1.0", log.NewNullLogger()).(*reporter)
	defer handler.Shutdown()

	t.Run("http", func(t *testing.T) {
		h := handler.InstrumentHttp("t", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasMetric("http.server.request.body.size"))
	})
	t.Run("http client", func(t *testing.T) {
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		client := http.Client{}
		client.Transport = handler.InstrumentHttpClient(http.DefaultTransport, NewKV("sdk", "test"))
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasMetric("http.client.request.body.size"))
	})
	t.Run("conn", func(t *testing.T) {
		handler.RecordConnections(5, "sdk", "grpc", "flag")

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasMetric("stream.connections"))
	})
	t.Run("sent msgs", func(t *testing.T) {
		handler.AddSentMessageCount(5, "sdk", "grpc", "flag")

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasMetric("stream.msg.sent.total"))
	})
	t.Run("redis", func(t *testing.T) {
		r := miniredis.RunT(t)
		opts := &redis.UniversalOptions{Addrs: []string{r.Addr()}}
		rdb := redis.NewUniversalClient(opts)
		defer func() { _ = rdb.Close() }()
		handler.InstrumentRedis(rdb)

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasMetric("db.client.connections.idle.max"))
	})
}

func TestHandler_Traces_Otlp_Export(t *testing.T) {
	collector, err := newInMemoryTraceGrpcCollector()
	assert.NoError(t, err)
	defer collector.Shutdown()
	assert.NoError(t, err)
	conf := config.DiagConfig{
		Port:    5052,
		Enabled: true,
		Traces:  config.TraceConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "grpc", Endpoint: collector.Addr()}},
	}

	handler := NewReporter(&conf, "0.1.0", log.NewDebugLogger())
	defer handler.Shutdown()

	t.Run("http", func(t *testing.T) {
		h := handler.InstrumentHttp("test", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasTrace("HTTP GET test"))
	})
	t.Run("http client", func(t *testing.T) {
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		defer srv.Close()
		client := http.Client{}
		client.Transport = handler.InstrumentHttpClient(http.DefaultTransport)
		ctx, span := handler.StartSpan(t.Context(), "test")
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)
		span.End()

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasTrace("test"))
	})
	t.Run("redis", func(t *testing.T) {
		r := miniredis.RunT(t)
		opts := &redis.UniversalOptions{Addrs: []string{r.Addr()}}
		rdb := redis.NewUniversalClient(opts)
		defer func() { _ = rdb.Close() }()
		handler.InstrumentRedis(rdb)
		rdb.Set(t.Context(), "test", "test", 0)

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasTrace("redis.dial"))
	})
	t.Run("mongo", func(t *testing.T) {
		mongodbContainer, err := mongodb.Run(t.Context(), "mongo")
		if err != nil {
			panic("failed to start container: " + err.Error() + "")
		}
		defer func() {
			if err := testcontainers.TerminateContainer(mongodbContainer); err != nil {
				panic("failed to terminate container: " + err.Error() + "")
			}
		}()
		str, _ := mongodbContainer.ConnectionString(t.Context())
		opts := options.Client().ApplyURI(str)
		handler.InstrumentMongoDb(opts)
		client, _ := mongo.Connect(opts)
		collection := client.Database("db").Collection("coll")
		_, _ = collection.InsertOne(t.Context(), map[string]interface{}{"test": "test"})

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasTrace("coll.insert"))
	})
	t.Run("dynamodb", func(t *testing.T) {
		dynamoDbContainer, err := tcdynamodb.Run(t.Context(), "amazon/dynamodb-local")
		if err != nil {
			panic("failed to start container: " + err.Error() + "")
		}
		defer func() {
			if err := testcontainers.TerminateContainer(dynamoDbContainer); err != nil {
				panic("failed to terminate container: " + err.Error() + "")
			}
		}()

		t.Setenv("AWS_ACCESS_KEY_ID", "key")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
		t.Setenv("AWS_SESSION_TOKEN", "session")
		t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

		awsCtx, err := awsconfig.LoadDefaultConfig(t.Context())
		handler.InstrumentAws(&awsCtx)
		str, _ := dynamoDbContainer.ConnectionString(t.Context())
		opts := []func(*dynamodb.Options){
			func(options *dynamodb.Options) {
				options.BaseEndpoint = aws.String("http://" + str)
			},
		}
		db := dynamodb.NewFromConfig(awsCtx, opts...)
		_, _ = db.DescribeTable(t.Context(), &dynamodb.DescribeTableInput{
			TableName: aws.String("test"),
		})

		handler.ForceFlush(t.Context())

		assert.True(t, collector.hasTrace("DynamoDB.DescribeTable"))
	})
}

func Test_Empty_Instrument(t *testing.T) {
	handler := NewEmptyReporter()

	assert.Nil(t, handler.InstrumentHttp("t", http.MethodGet, nil))
	assert.Equal(t, http.DefaultTransport, handler.InstrumentHttpClient(http.DefaultTransport))
	assert.Empty(t, handler.InstrumentGrpc([]grpc.ServerOption{}))
	_, span := handler.StartSpan(t.Context(), "t")
	assert.Equal(t, noop.Span{}, span)

	handler.ForceFlush(t.Context())
	handler.RecordConnections(5, "sdk", "grpc", "flag")
	handler.AddSentMessageCount(5, "sdk", "grpc", "flag")
	handler.InstrumentRedis(nil)

	opts := &options.ClientOptions{}
	handler.InstrumentMongoDb(opts)
	assert.Nil(t, opts.Monitor)

	conf := &aws.Config{}
	handler.InstrumentAws(conf)
	assert.Empty(t, conf.APIOptions)

	handler.Shutdown()
}
