package config

import (
	"errors"
	"fmt"
	"os"
)

func (c *Config) Validate() error {
	if len(c.SDKs) == 0 && !c.AutoSDK.IsSet() {
		return fmt.Errorf("sdk: at least 1 SDK must be configured")
	}
	if c.AutoSDK.IsSet() {
		if err := c.AutoSDK.validate(); err != nil {
			return err
		}
	}
	for id, conf := range c.SDKs {
		if err := conf.validate(&c.Cache, id); err != nil {
			return err
		}
	}
	if err := c.Tls.validate(); err != nil {
		return err
	}
	if err := c.Http.validate(); err != nil {
		return err
	}
	if err := c.Diag.validate(); err != nil {
		return err
	}
	if err := c.Grpc.validate(); err != nil {
		return err
	}
	if err := c.Cache.Redis.validate(); err != nil {
		return err
	}
	if err := c.Cache.MongoDb.validate(); err != nil {
		return err
	}
	if err := c.GlobalOfflineConfig.validate(&c.Cache); err != nil {
		return err
	}
	return nil
}

func (s *SDKConfig) validate(c *CacheConfig, sdkId string) error {
	if s.Key == "" {
		return fmt.Errorf("sdk-%s: SDK key is required", sdkId)
	}
	if s.DataGovernance != "" && s.DataGovernance != "global" && s.DataGovernance != "eu" {
		return fmt.Errorf("sdk-%s: invalid data governance value, it must be 'global' or 'eu'", sdkId)
	}
	if s.WebhookSigningKey != "" && s.WebhookSignatureValidFor < 5 {
		return fmt.Errorf("sdk-%s: webhook signature validity check must be greater than 5 seconds", sdkId)
	}
	if err := s.Offline.validate(c, sdkId); err != nil {
		return err
	}
	return nil
}

func (r *RedisConfig) validate() error {
	if !r.Enabled {
		return nil
	}
	if len(r.Addresses) == 0 {
		return fmt.Errorf("redis: at least 1 server address required")
	}
	if err := r.Tls.validate(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDbConfig) validate() error {
	if !m.Enabled {
		return nil
	}
	if len(m.Url) == 0 {
		return fmt.Errorf("mongodb: invalid connection uri")
	}
	if err := m.Tls.validate(); err != nil {
		return err
	}
	return nil
}

func (o *OfflineConfig) validate(c *CacheConfig, sdkId string) error {
	if !o.Enabled {
		return nil
	}
	if o.Local.FilePath == "" && !o.UseCache {
		return fmt.Errorf("sdk-%s: offline mode requires either a configured cache or a local file", sdkId)
	}
	if o.Local.FilePath != "" && o.UseCache {
		return fmt.Errorf("sdk-%s: can't use both local file and cache for offline mode", sdkId)
	}
	if o.Local.FilePath != "" {
		if err := o.Local.validate(sdkId); err != nil {
			return err
		}
	}
	if o.UseCache && !c.IsSet() {
		return fmt.Errorf("sdk-%s: offline mode enabled with cache, but no cache is configured", sdkId)
	}
	if o.UseCache && o.CachePollInterval < 1 {
		return fmt.Errorf("sdk-%s: cache poll interval must be greater than 1 seconds", sdkId)
	}
	return nil
}

func (g *GlobalOfflineConfig) validate(c *CacheConfig) error {
	if !g.Enabled {
		return nil
	}
	if !c.Redis.Enabled {
		return fmt.Errorf("offline: global offline mode enabled, but no cache is configured")
	}
	if g.CachePollInterval < 1 {
		return fmt.Errorf("offline: cache poll interval must be greater than 1 seconds")
	}
	return nil
}

func (t *TlsConfig) validate() error {
	if !t.Enabled {
		return nil
	}
	for _, cert := range t.Certificates {
		if (cert.Cert != "" && cert.Key == "") || (cert.Key != "" && cert.Cert == "") {
			return fmt.Errorf("tls: both TLS cert and key file required")
		}
	}
	return nil
}

func (h *HttpConfig) validate() error {
	if h.Port < 1 || h.Port > 65535 {
		return fmt.Errorf("http: invalid port %d", h.Port)
	}
	if err := h.Webhook.validate(); err != nil {
		return err
	}
	if err := h.Api.CORS.validate(); err != nil {
		return err
	}
	if err := h.OFREP.CORS.validate(); err != nil {
		return err
	}
	if err := h.Sse.CORS.validate(); err != nil {
		return err
	}
	if err := h.CdnProxy.CORS.validate(); err != nil {
		return err
	}
	return nil
}

func (w *WebhookConfig) validate() error {
	if !w.Enabled {
		return nil
	}
	if (w.Auth.User != "" && w.Auth.Password == "") || (w.Auth.Password != "" && w.Auth.User == "") {
		return fmt.Errorf("webhook: both basic auth user and password required")
	}
	return nil
}

func (c *CORSConfig) validate() error {
	if !c.Enabled {
		return nil
	}
	if err := c.AllowedOriginsRegex.validate(); err != nil {
		return err
	}
	return nil
}

func (o *OriginRegexConfig) validate() error {
	if o.Patterns != nil && len(o.Patterns) > 0 && o.IfNoMatch == "" {
		return fmt.Errorf("cors: the 'if no watch' field is required when allowed origins regex is set")
	}
	return nil
}

func (l *LocalConfig) validate(sdkId string) error {
	if _, err := os.Stat(l.FilePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("sdk-"+sdkId+": couldn't find the local file %s", l.FilePath)
	}
	if l.Polling && l.PollInterval < 1 {
		return fmt.Errorf("sdk-%s: local file poll interval must be greater than 1 seconds", sdkId)
	}
	return nil
}

func (d *DiagConfig) validate() error {
	if d.Port < 1 || d.Port > 65535 {
		return fmt.Errorf("diag: invalid port %d", d.Port)
	}
	return nil
}

func (g *GrpcConfig) validate() error {
	if g.Port < 1 || g.Port > 65535 {
		return fmt.Errorf("grpc: invalid port %d", g.Port)
	}
	return nil
}

func (a *AutoSDKConfig) validate() error {
	if a.PollInterval < 30 {
		return fmt.Errorf("sdk: auto configuration poll interval cannot be less than 30 seconds")
	}
	return nil
}
