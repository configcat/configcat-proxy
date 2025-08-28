package mware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORS(t *testing.T) {
	t.Run("* origin, options", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, nil, nil, nil, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, options", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, []string{"http://localhost"}, nil, nil, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		req.Header.Set("Origin", "http://localhost")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "http://localhost", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("* origin, get", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, nil, nil, nil, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, get", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, []string{"http://localhost"}, nil, nil, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Origin", "http://localhost")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "http://localhost", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, options, multiple origins", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, []string{"https://test1.com", "https://test2.com"}, []string{"h1", "ETag"}, []string{"X-AUTH"}, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		req.Header.Set("Origin", "https://test1.com")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		req.Header.Set("Origin", "https://test2.com")
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "https://test2.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		req.Header.Set("Origin", "something-else")
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, get, multiple origins", func(t *testing.T) {
		handler := CORS([]string{http.MethodGet, http.MethodOptions}, []string{"https://test1.com", "https://test2.com"}, nil, nil, nil, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(handler)
		client := http.Client{}

		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Origin", "https://test1.com")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Origin", "https://test2.com")
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "https://test2.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Origin", "something-else")
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))

		req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ = client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, get, regex", func(t *testing.T) {
		regex1, _ := regexp.Compile(".*test1\\.com")
		regex2, _ := regexp.Compile(".*test2\\.com")
		t.Run("only regex", func(t *testing.T) {
			handler := CORS([]string{http.MethodGet, http.MethodOptions}, nil, nil, nil, &config.OriginRegexConfig{
				Regexes: []*regexp.Regexp{
					regex1,
					regex2,
				},
				IfNoMatch: "https://test3.com",
			}, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusOK)
			})
			srv := httptest.NewServer(handler)
			client := http.Client{}

			req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://test1.com")
			resp, _ := client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test1.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test2.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test2.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.someelse.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test3.com", resp.Header.Get("Access-Control-Allow-Origin"))
		})
		t.Run("both", func(t *testing.T) {
			handler := CORS([]string{http.MethodGet, http.MethodOptions}, []string{"https://test3.com", "https://test4.com"}, nil, nil, &config.OriginRegexConfig{
				Regexes: []*regexp.Regexp{
					regex1,
					regex2,
				},
				IfNoMatch: "https://test5.com",
			}, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusOK)
			})
			srv := httptest.NewServer(handler)
			client := http.Client{}

			req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://test1.com")
			resp, _ := client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test1.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test2.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test2.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://test3.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test3.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://test4.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test4.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.someelse.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test5.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test5.com", resp.Header.Get("Access-Control-Allow-Origin"))
		})
	})
	t.Run("custom origin, get, from config", func(t *testing.T) {
		testutils.UseTempFile(`
http:
  api:
    cors: 
      enabled: true
      allowed_origins_regex:
        patterns:
          - .*test1\.com
          - .*test2\.com
        if_no_match: https://test3.com
`, func(file string) {
			conf, err := config.LoadConfigFromFileAndEnvironment(file)
			require.NoError(t, err)

			handler := CORS([]string{http.MethodGet, http.MethodOptions}, conf.Http.Api.CORS.AllowedOrigins, nil, nil, &conf.Http.Api.CORS.AllowedOriginsRegex, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusOK)
			})
			srv := httptest.NewServer(handler)
			client := http.Client{}

			req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://test1.com")
			resp, _ := client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test1.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test1.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.test2.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://sub1.test2.com", resp.Header.Get("Access-Control-Allow-Origin"))

			req, _ = http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
			req.Header.Set("Origin", "https://sub1.someelse.com")
			resp, _ = client.Do(req)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "https://test3.com", resp.Header.Get("Access-Control-Allow-Origin"))
		})
	})
}
