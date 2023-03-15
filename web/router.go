package web

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
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
	metrics        metrics.Handler
}

func NewRouter(sdkClients map[string]sdk.Client, metrics metrics.Handler, reporter status.Reporter, conf *config.HttpConfig, log log.Logger) *HttpRouter {
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
	r.setupStatusRoutes(reporter)
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

func (s *HttpRouter) setupSSERoutes(conf *config.SseConfig, sdkClients map[string]sdk.Client, log log.Logger) {
	s.sseServer = sse.NewServer(sdkClients, s.metrics, conf, log)
	path := "/sse/:env/:data"
	handler := mware.AutoOptions(s.sseServer.ServeHTTP)
	if len(conf.Headers) > 0 {
		handler = mware.ExtraHeaders(conf.Headers, handler)
	}
	if conf.AllowCORS {
		handler = mware.CORS([]string{http.MethodGet, http.MethodOptions}, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodOptions, path, handler)
	log.Reportf("SSE enabled, listening on path: %s", path)
}

func (s *HttpRouter) setupWebhookRoutes(conf *config.WebhookConfig, sdkClients map[string]sdk.Client, log log.Logger) {
	s.webhookServer = webhook.NewServer(sdkClients, log)
	path := "/hook/:env"
	handler := http.HandlerFunc(s.webhookServer.ServeHTTP)
	if conf.Auth.User != "" && conf.Auth.Password != "" {
		handler = mware.BasicAuth(conf.Auth.User, conf.Auth.Password, log, handler)
	}
	if len(conf.AuthHeaders) > 0 {
		handler = mware.HeaderAuth(conf.AuthHeaders, log, handler)
	}
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodPost, path, handler)
	log.Reportf("webhook enabled, accepting requests on path: %s", path)
}

func (s *HttpRouter) setupCDNProxyRoutes(conf *config.CdnProxyConfig, sdkClients map[string]sdk.Client, log log.Logger) {
	s.cdnProxyServer = cdnproxy.NewServer(sdkClients, conf, log)
	path := "/configuration-files/:env/:file"
	handler := mware.AutoOptions(mware.GZip(s.cdnProxyServer.ServeHTTP))
	if len(conf.Headers) > 0 {
		handler = mware.ExtraHeaders(conf.Headers, handler)
	}
	if conf.AllowCORS {
		handler = mware.CORS([]string{http.MethodGet, http.MethodOptions}, handler)
	}
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodOptions, path, handler)
	log.Reportf("CDN proxy enabled, accepting requests on path: %s", path)
}

func (s *HttpRouter) setupStatusRoutes(reporter status.Reporter) {
	path := "/status"
	handler := mware.AutoOptions(mware.GZip(reporter.HttpHandler()))
	if s.metrics != nil {
		handler = metrics.Measure(s.metrics, handler)
	}
	s.router.HandlerFunc(http.MethodGet, path, handler)
	s.router.HandlerFunc(http.MethodOptions, path, handler)
}

type endpoint struct {
	handler http.HandlerFunc
	method  string
	path    string
}

func (s *HttpRouter) setupAPIRoutes(conf *config.ApiConfig, sdkClients map[string]sdk.Client, log log.Logger) {
	s.apiServer = api.NewServer(sdkClients, conf, log)
	endpoints := []endpoint{
		{path: "/api/:env/eval", handler: mware.GZip(s.apiServer.Eval), method: http.MethodPost},
		{path: "/api/:env/eval-all", handler: mware.GZip(s.apiServer.EvalAll), method: http.MethodPost},
		{path: "/api/:env/keys", handler: mware.GZip(s.apiServer.Keys), method: http.MethodGet},
		{path: "/api/:env/refresh", handler: http.HandlerFunc(s.apiServer.Refresh), method: http.MethodPost},
	}
	for _, endpoint := range endpoints {
		if len(conf.AuthHeaders) > 0 {
			endpoint.handler = mware.HeaderAuth(conf.AuthHeaders, log, endpoint.handler)
		}
		endpoint.handler = mware.AutoOptions(endpoint.handler)
		if len(conf.Headers) > 0 {
			endpoint.handler = mware.ExtraHeaders(conf.Headers, endpoint.handler)
		}
		if conf.AllowCORS {
			endpoint.handler = mware.CORS([]string{endpoint.method, http.MethodOptions}, endpoint.handler)
		}
		if s.metrics != nil {
			endpoint.handler = metrics.Measure(s.metrics, endpoint.handler)
		}
		s.router.HandlerFunc(endpoint.method, endpoint.path, endpoint.handler)
		s.router.HandlerFunc(http.MethodOptions, endpoint.path, endpoint.handler)
	}
	log.Reportf("API enabled, accepting requests on path: /api/:env/*")
}
