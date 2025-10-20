package web

import (
	"net/http"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/web/api"
	"github.com/configcat/configcat-proxy/web/cdnproxy"
	"github.com/configcat/configcat-proxy/web/mware"
	"github.com/configcat/configcat-proxy/web/ofrep"
	"github.com/configcat/configcat-proxy/web/sse"
	"github.com/configcat/configcat-proxy/web/webhook"
)

type HttpRouter struct {
	router            *http.ServeMux
	sseServer         *sse.Server
	webhookServer     *webhook.Server
	cdnProxyServer    *cdnproxy.Server
	apiServer         *api.Server
	ofrepServer       *ofrep.Server
	telemetryReporter telemetry.Reporter
}

func NewRouter(sdkRegistrar sdk.Registrar, telemetryReporter telemetry.Reporter, reporter status.Reporter, conf *config.HttpConfig, autoSdkConfig *config.ProfileConfig, log log.Logger) *HttpRouter {
	httpLog := log.WithLevel(conf.Log.GetLevel()).WithPrefix("http")

	r := &HttpRouter{
		router:            http.NewServeMux(),
		telemetryReporter: telemetryReporter,
	}
	if conf.Sse.Enabled {
		r.setupSSERoutes(&conf.Sse, sdkRegistrar, httpLog)
	}
	if conf.Webhook.Enabled {
		r.setupWebhookRoutes(&conf.Webhook, autoSdkConfig, sdkRegistrar, httpLog)
	}
	if conf.CdnProxy.Enabled {
		r.setupCDNProxyRoutes(&conf.CdnProxy, sdkRegistrar, httpLog)
	}
	if conf.Api.Enabled {
		r.setupAPIRoutes(&conf.Api, sdkRegistrar, httpLog)
	}
	if conf.OFREP.Enabled {
		r.setupOFREPRoutes(&conf.OFREP, sdkRegistrar, httpLog)
	}
	if conf.Status.Enabled {
		r.setupStatusRoutes(reporter, httpLog)
	}
	return r
}

func (s *HttpRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if req.Method != http.MethodConnect && path != "/" && len(path) > 1 && path[len(path)-1] == '/' {
		req.URL.Path = path[:len(path)-1]
	}
	s.router.ServeHTTP(w, req)
}

func (s *HttpRouter) Close() {
	if s.sseServer != nil {
		s.sseServer.Close()
	}
}

func (s *HttpRouter) setupSSERoutes(conf *config.SseConfig, sdkRegistrar sdk.Registrar, l log.Logger) {
	s.sseServer = sse.NewServer(sdkRegistrar, s.telemetryReporter, conf, l)
	endpoints := []endpoint{
		{path: "/sse/{sdkId}/eval/{data}", handler: http.HandlerFunc(s.sseServer.SingleFlag), method: http.MethodGet},
		{path: "/sse/{sdkId}/eval-all/{data}", handler: http.HandlerFunc(s.sseServer.AllFlags), method: http.MethodGet},
		{path: "/sse/{sdkId}/eval-all", handler: http.HandlerFunc(s.sseServer.AllFlags), method: http.MethodGet},
		{path: "/sse/eval/k/{data}", handler: http.HandlerFunc(s.sseServer.SingleFlag), method: http.MethodGet},
		{path: "/sse/eval-all/k/{data}", handler: http.HandlerFunc(s.sseServer.AllFlags), method: http.MethodGet},
	}
	for _, endpoint := range endpoints {
		endpoint.handler = mware.AutoOptions(endpoint.handler)
		if len(conf.Headers) > 0 {
			endpoint.handler = mware.ExtraHeaders(conf.Headers, endpoint.handler)
		}
		if conf.CORS.Enabled {
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, conf.CORS.AllowedOrigins,
				utils.KeysOfMap(conf.Headers), nil, &conf.CORS.AllowedOriginsRegex, endpoint.handler)
		}
		if l.Level() == log.Debug {
			endpoint.handler = mware.DebugLog(l, endpoint.handler)
		}
		s.router.HandleFunc(addHttpMethod(endpoint.path, endpoint.method), endpoint.handler)
		s.router.HandleFunc(addHttpMethod(endpoint.path, http.MethodOptions), endpoint.handler)
	}
	l.Reportf("SSE enabled, accepting requests on path: /sse/*")
}

func (s *HttpRouter) setupWebhookRoutes(conf *config.WebhookConfig, autoSdkConfig *config.ProfileConfig, sdkRegistrar sdk.Registrar, l log.Logger) {
	s.webhookServer = webhook.NewServer(autoSdkConfig, sdkRegistrar, l)
	path := "/hook/{sdkId}"
	testPath := "/hook-test"
	handler := http.HandlerFunc(s.webhookServer.ServeWebhookSdkId)
	testHandler := http.HandlerFunc(s.webhookServer.ServeWebhookTest)
	if conf.Auth.User != "" && conf.Auth.Password != "" {
		handler = mware.BasicAuth(conf.Auth.User, conf.Auth.Password, l, handler)
	}
	if len(conf.AuthHeaders) > 0 {
		handler = mware.HeaderAuth(conf.AuthHeaders, l, handler)
	}
	if l.Level() == log.Debug {
		handler = mware.DebugLog(l, handler)
		testHandler = mware.DebugLog(l, testHandler)
	}
	s.router.HandleFunc(addHttpMethod(path, http.MethodGet), s.telemetryReporter.InstrumentHttp(path, http.MethodGet, handler))
	s.router.HandleFunc(addHttpMethod(path, http.MethodPost), s.telemetryReporter.InstrumentHttp(path, http.MethodPost, handler))
	s.router.HandleFunc(addHttpMethod(testPath, http.MethodGet), s.telemetryReporter.InstrumentHttp(path, http.MethodGet, testHandler))
	s.router.HandleFunc(addHttpMethod(testPath, http.MethodPost), s.telemetryReporter.InstrumentHttp(path, http.MethodPost, testHandler))
	l.Reportf("webhook enabled, accepting requests on path: %s", path)
}

func (s *HttpRouter) setupCDNProxyRoutes(conf *config.CdnProxyConfig, sdkRegistrar sdk.Registrar, l log.Logger) {
	s.cdnProxyServer = cdnproxy.NewServer(sdkRegistrar, conf, l)
	path := "/configuration-files/{path...}"
	handler := mware.AutoOptions(mware.GZip(s.cdnProxyServer.ServeHTTP))
	if len(conf.Headers) > 0 {
		handler = mware.ExtraHeaders(conf.Headers, handler)
	}
	if conf.CORS.Enabled {
		handler = mware.CORS([]string{http.MethodGet, http.MethodOptions}, conf.CORS.AllowedOrigins,
			utils.KeysOfMap(conf.Headers), nil, &conf.CORS.AllowedOriginsRegex, handler)
	}
	if l.Level() == log.Debug {
		handler = mware.DebugLog(l, handler)
	}
	s.router.HandleFunc(addHttpMethod(path, http.MethodGet), s.telemetryReporter.InstrumentHttp(path, http.MethodGet, handler))
	s.router.HandleFunc(addHttpMethod(path, http.MethodOptions), s.telemetryReporter.InstrumentHttp(path, http.MethodOptions, handler))
	l.Reportf("CDN proxy enabled, accepting requests on path: %s", path)
}

func (s *HttpRouter) setupStatusRoutes(reporter status.Reporter, l log.Logger) {
	path := "/status"
	handler := mware.AutoOptions(mware.GZip(reporter.HttpHandler()))
	s.router.HandleFunc(addHttpMethod(path, http.MethodGet), handler)
	s.router.HandleFunc(addHttpMethod(path, http.MethodOptions), handler)
	l.Reportf("status enabled, accepting requests on path: %s", path)
}

type endpoint struct {
	handler http.HandlerFunc
	method  string
	path    string
}

func (s *HttpRouter) setupAPIRoutes(conf *config.ApiConfig, sdkRegistrar sdk.Registrar, l log.Logger) {
	s.apiServer = api.NewServer(sdkRegistrar, conf, l)
	endpoints := []endpoint{
		{path: "/api/{sdkId}/eval", handler: mware.GZip(s.apiServer.Eval), method: http.MethodPost},
		{path: "/api/{sdkId}/eval-all", handler: mware.GZip(s.apiServer.EvalAll), method: http.MethodPost},
		{path: "/api/{sdkId}/keys", handler: mware.GZip(s.apiServer.Keys), method: http.MethodGet},
		{path: "/api/{sdkId}/refresh", handler: http.HandlerFunc(s.apiServer.Refresh), method: http.MethodPost},
		{path: "/api/eval", handler: mware.GZip(s.apiServer.Eval), method: http.MethodPost},
		{path: "/api/eval-all", handler: mware.GZip(s.apiServer.EvalAll), method: http.MethodPost},
		{path: "/api/keys", handler: mware.GZip(s.apiServer.Keys), method: http.MethodGet},
		{path: "/api/refresh", handler: http.HandlerFunc(s.apiServer.Refresh), method: http.MethodPost},
		{path: "/api/icanhascoffee", handler: http.HandlerFunc(s.apiServer.ICanHasCoffee), method: http.MethodGet},
	}
	for _, endpoint := range endpoints {
		if len(conf.AuthHeaders) > 0 {
			endpoint.handler = mware.HeaderAuth(conf.AuthHeaders, l, endpoint.handler)
		}
		endpoint.handler = mware.AutoOptions(endpoint.handler)
		if len(conf.Headers) > 0 {
			endpoint.handler = mware.ExtraHeaders(conf.Headers, endpoint.handler)
		}
		if conf.CORS.Enabled {
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, conf.CORS.AllowedOrigins,
				utils.KeysOfMap(conf.Headers), utils.KeysOfMap(conf.AuthHeaders), &conf.CORS.AllowedOriginsRegex, endpoint.handler)
		}
		if l.Level() == log.Debug {
			endpoint.handler = mware.DebugLog(l, endpoint.handler)
		}
		s.router.HandleFunc(addHttpMethod(endpoint.path, endpoint.method), s.telemetryReporter.InstrumentHttp(endpoint.path, endpoint.method, endpoint.handler))
		s.router.HandleFunc(addHttpMethod(endpoint.path, http.MethodOptions), s.telemetryReporter.InstrumentHttp(endpoint.path, http.MethodOptions, endpoint.handler))
	}
	l.Reportf("API enabled, accepting requests on path: /api/*")
}

func (s *HttpRouter) setupOFREPRoutes(conf *config.OFREPConfig, sdkRegistrar sdk.Registrar, l log.Logger) {
	s.ofrepServer = ofrep.NewServer(sdkRegistrar, conf, l)
	endpoints := []endpoint{
		{path: "/ofrep/v1/evaluate/flags/{key}", handler: mware.GZip(s.ofrepServer.Eval), method: http.MethodPost},
		{path: "/ofrep/v1/evaluate/flags", handler: mware.GZip(s.ofrepServer.EvalAll), method: http.MethodPost},
	}
	for _, endpoint := range endpoints {
		if len(conf.AuthHeaders) > 0 {
			endpoint.handler = mware.HeaderAuth(conf.AuthHeaders, l, endpoint.handler)
		}
		endpoint.handler = mware.AutoOptions(endpoint.handler)
		if len(conf.Headers) > 0 {
			endpoint.handler = mware.ExtraHeaders(conf.Headers, endpoint.handler)
		}
		if conf.CORS.Enabled {
			allowedHeaders := append(utils.KeysOfMap(conf.AuthHeaders), ofrep.SdkIdHeader)
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, conf.CORS.AllowedOrigins,
				utils.KeysOfMap(conf.Headers), allowedHeaders, &conf.CORS.AllowedOriginsRegex, endpoint.handler)
		}
		if l.Level() == log.Debug {
			endpoint.handler = mware.DebugLog(l, endpoint.handler)
		}
		s.router.HandleFunc(addHttpMethod(endpoint.path, endpoint.method), s.telemetryReporter.InstrumentHttp(endpoint.path, endpoint.method, endpoint.handler))
		s.router.HandleFunc(addHttpMethod(endpoint.path, http.MethodOptions), s.telemetryReporter.InstrumentHttp(endpoint.path, http.MethodOptions, endpoint.handler))
	}
	l.Reportf("OFREP enabled, accepting requests on path: /ofrep/v1/evaluate/flags/*")
}

func addHttpMethod(path string, method string) string {
	return method + " " + path
}
