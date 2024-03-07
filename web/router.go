package web

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/web/api"
	"github.com/configcat/configcat-proxy/web/cdnproxy"
	"github.com/configcat/configcat-proxy/web/mware"
	"github.com/configcat/configcat-proxy/web/sse"
	"github.com/configcat/configcat-proxy/web/webhook"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type HttpRouter struct {
	router         *httprouter.Router
	sseServer      *sse.Server
	webhookServer  *webhook.Server
	cdnProxyServer *cdnproxy.Server
	apiServer      *api.Server
	metrics        metrics.Reporter
}

func NewRouter(sdkClients map[string]sdk.Client, metrics metrics.Reporter, reporter status.Reporter, conf *config.HttpConfig, log log.Logger) *HttpRouter {
	httpLog := log.WithLevel(conf.Log.GetLevel()).WithPrefix("http")

	r := &HttpRouter{
		router: &httprouter.Router{
			RedirectFixedPath:      true,
			RedirectTrailingSlash:  true,
			HandleMethodNotAllowed: true,
		},
		metrics: metrics,
	}
	if conf.Sse.Enabled {
		r.setupSSERoutes(&conf.Sse, sdkClients, httpLog)
	}
	if conf.Webhook.Enabled {
		r.setupWebhookRoutes(&conf.Webhook, sdkClients, httpLog)
	}
	if conf.CdnProxy.Enabled {
		r.setupCDNProxyRoutes(&conf.CdnProxy, sdkClients, httpLog)
	}
	if conf.Api.Enabled {
		r.setupAPIRoutes(&conf.Api, sdkClients, httpLog)
	}
	if conf.Status.Enabled {
		r.setupStatusRoutes(reporter, httpLog)
	}
	return r
}

func (s *HttpRouter) Handler() http.Handler {
	return s.router
}

func (s *HttpRouter) Close() {
	if s.sseServer != nil {
		s.sseServer.Close()
	}
}

func (s *HttpRouter) setupSSERoutes(conf *config.SseConfig, sdkClients map[string]sdk.Client, l log.Logger) {
	s.sseServer = sse.NewServer(sdkClients, s.metrics, conf, l)
	endpoints := []endpoint{
		{path: "/sse/:sdkId/eval/:data", handler: http.HandlerFunc(s.sseServer.SingleFlag), method: http.MethodGet},
		{path: "/sse/:sdkId/eval-all/:data", handler: http.HandlerFunc(s.sseServer.AllFlags), method: http.MethodGet},
		{path: "/sse/:sdkId/eval-all/", handler: http.HandlerFunc(s.sseServer.AllFlags), method: http.MethodGet},
	}
	for _, endpoint := range endpoints {
		endpoint.handler = mware.AutoOptions(endpoint.handler)
		if len(conf.Headers) > 0 {
			endpoint.handler = mware.ExtraHeaders(conf.Headers, endpoint.handler)
		}
		if conf.CORS.Enabled {
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, conf.CORS.AllowedOrigins, utils.Keys(conf.Headers), nil, &conf.CORS.AllowedOriginsRegex, endpoint.handler)
		}
		if l.Level() == log.Debug {
			endpoint.handler = mware.DebugLog(l, endpoint.handler)
		}
		s.router.HandlerFunc(endpoint.method, endpoint.path, endpoint.handler)
		s.router.HandlerFunc(http.MethodOptions, endpoint.path, endpoint.handler)
	}
	l.Reportf("SSE enabled, accepting requests on path: /sse/:sdkId/*")
}

func (s *HttpRouter) setupWebhookRoutes(conf *config.WebhookConfig, sdkClients map[string]sdk.Client, l log.Logger) {
	s.webhookServer = webhook.NewServer(sdkClients, l)
	path := "/hook/:sdkId"
	handler := http.HandlerFunc(s.webhookServer.ServeHTTP)
	if conf.Auth.User != "" && conf.Auth.Password != "" {
		handler = mware.BasicAuth(conf.Auth.User, conf.Auth.Password, l, handler)
	}
	if len(conf.AuthHeaders) > 0 {
		handler = mware.HeaderAuth(conf.AuthHeaders, l, handler)
	}
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	if l.Level() == log.Debug {
		handler = mware.DebugLog(l, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodPost, path, handler)
	l.Reportf("webhook enabled, accepting requests on path: %s", path)
}

func (s *HttpRouter) setupCDNProxyRoutes(conf *config.CdnProxyConfig, sdkClients map[string]sdk.Client, l log.Logger) {
	s.cdnProxyServer = cdnproxy.NewServer(sdkClients, conf, l)
	path := "/configuration-files/configcat-proxy/:sdkId/config_v6.json"
	handler := mware.AutoOptions(mware.GZip(s.cdnProxyServer.ServeHTTP))
	if len(conf.Headers) > 0 {
		handler = mware.ExtraHeaders(conf.Headers, handler)
	}
	if conf.CORS.Enabled {
		handler = mware.CORS([]string{http.MethodGet, http.MethodOptions}, conf.CORS.AllowedOrigins, utils.Keys(conf.Headers), nil, &conf.CORS.AllowedOriginsRegex, handler)
	}
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	if l.Level() == log.Debug {
		handler = mware.DebugLog(l, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodOptions, path, handler)
	l.Reportf("CDN proxy enabled, accepting requests on paths: %s", path)
}

func (s *HttpRouter) setupStatusRoutes(reporter status.Reporter, l log.Logger) {
	path := "/status"
	handler := mware.AutoOptions(mware.GZip(reporter.HttpHandler()))
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodOptions, path, handler)
	l.Reportf("status enabled, accepting requests on path: %s", path)
}

type endpoint struct {
	handler http.HandlerFunc
	method  string
	path    string
}

func (s *HttpRouter) setupAPIRoutes(conf *config.ApiConfig, sdkClients map[string]sdk.Client, l log.Logger) {
	s.apiServer = api.NewServer(sdkClients, conf, l)
	endpoints := []endpoint{
		{path: "/api/:sdkId/eval", handler: mware.GZip(s.apiServer.Eval), method: http.MethodPost},
		{path: "/api/:sdkId/eval-all", handler: mware.GZip(s.apiServer.EvalAll), method: http.MethodPost},
		{path: "/api/:sdkId/keys", handler: mware.GZip(s.apiServer.Keys), method: http.MethodGet},
		{path: "/api/:sdkId/refresh", handler: http.HandlerFunc(s.apiServer.Refresh), method: http.MethodPost},
		{path: "/api/:sdkId/icanhascoffee", handler: http.HandlerFunc(s.apiServer.ICanHasCoffee), method: http.MethodGet},
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
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, conf.CORS.AllowedOrigins, utils.Keys(conf.Headers), utils.Keys(conf.AuthHeaders), &conf.CORS.AllowedOriginsRegex, endpoint.handler)
		}
		if s.metrics != nil {
			endpoint.handler = metrics.Measure(s.metrics, endpoint.handler)
		}
		if l.Level() == log.Debug {
			endpoint.handler = mware.DebugLog(l, endpoint.handler)
		}
		s.router.HandlerFunc(endpoint.method, endpoint.path, endpoint.handler)
		s.router.HandlerFunc(http.MethodOptions, endpoint.path, endpoint.handler)
	}
	l.Reportf("API enabled, accepting requests on path: /api/:sdkId/*")
}
