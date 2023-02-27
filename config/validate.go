package config

import (
	"errors"
	"fmt"
	"os"
)

func (c *Config) Validate() error {
	if err := c.SDK.validate(); err != nil {
		return err
	}
	if err := c.Tls.validate(); err != nil {
		return err
	}
	if err := c.Http.Webhook.validate(); err != nil {
		return err
	}
	return nil
}

func (s *SDKConfig) validate() error {
	if s.Key == "" {
		return fmt.Errorf("sdk: SDK key is required")
	}
	if s.DataGovernance != "" && s.DataGovernance != "global" && s.DataGovernance != "eu" {
		return fmt.Errorf("sdk: invalid data governance value, it must be 'global' or 'eu'")
	}
	if err := s.Offline.validate(&s.Cache); err != nil {
		return err
	}
	if err := s.Cache.Redis.validate(); err != nil {
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

func (o *OfflineConfig) validate(c *CacheConfig) error {
	if !o.Enabled {
		return nil
	}
	if o.Local.FilePath == "" && !o.UseCache {
		return fmt.Errorf("sdk: offline mode requires either a configured cache or a local file")
	}
	if o.Local.FilePath != "" && o.UseCache {
		return fmt.Errorf("sdk: can't use both local file and cache for offline mode")
	}
	if o.Local.FilePath != "" {
		if err := o.Local.validate(); err != nil {
			return err
		}
	}
	if o.UseCache && !c.Redis.Enabled {
		return fmt.Errorf("sdk: offline mode enabled with cache, but no cache is configured")
	}
	if o.UseCache && o.CachePollInterval < 1 {
		return fmt.Errorf("sdk: cache poll interval must be greater than 1 seconds")
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

func (w *WebhookConfig) validate() error {
	if !w.Enabled {
		return nil
	}
	if w.SigningKey != "" && w.SignatureValidFor < 5 {
		return fmt.Errorf("webhook: signature validity check must be greater than 5 seconds")
	}
	if (w.Auth.User != "" && w.Auth.Password == "") || (w.Auth.Password != "" && w.Auth.User == "") {
		return fmt.Errorf("webhook: both basic auth user and password required")
	}
	return nil
}

func (l *LocalConfig) validate() error {
	if _, err := os.Stat(l.FilePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("sdk: couldn't find the local file %s", l.FilePath)
	}
	if l.Polling && l.PollInterval < 1 {
		return fmt.Errorf("sdk: local file poll interval must be greater than 1 seconds")
	}
	return nil
}
