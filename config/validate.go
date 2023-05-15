package config

import (
	"errors"
	"fmt"
	"os"
)

func (c *Config) Validate() error {
	if len(c.SDKs) == 0 {
		return fmt.Errorf("sdk: at least 1 environment with an SDK key required")
	}
	for id, env := range c.SDKs {
		if err := env.validate(&c.Cache, id); err != nil {
			return err
		}
	}
	if err := c.Tls.validate(); err != nil {
		return err
	}
	if err := c.Http.Webhook.validate(); err != nil {
		return err
	}
	if err := c.Cache.Redis.validate(); err != nil {
		return err
	}
	if err := c.EvalStats.InfluxDb.validate(); err != nil {
		return err
	}
	return nil
}

func (s *SDKConfig) validate(c *CacheConfig, envId string) error {
	if s.Key == "" {
		return fmt.Errorf("sdk-" + envId + ": SDK key is required")
	}
	if s.DataGovernance != "" && s.DataGovernance != "global" && s.DataGovernance != "eu" {
		return fmt.Errorf("sdk-" + envId + ": invalid data governance value, it must be 'global' or 'eu'")
	}
	if s.WebhookSigningKey != "" && s.WebhookSignatureValidFor < 5 {
		return fmt.Errorf("sdk-" + envId + ": webhook signature validity check must be greater than 5 seconds")
	}
	if err := s.Offline.validate(c, envId); err != nil {
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

func (i *InfluxDbConfig) validate() error {
	if !i.Enabled {
		return nil
	}
	if i.Url == "" {
		return fmt.Errorf("influxdb: URL is required")
	}
	if i.Organization == "" {
		return fmt.Errorf("influxdb: organization is required")
	}
	if i.Bucket == "" {
		return fmt.Errorf("influxdb: bucket is required")
	}
	if i.AuthToken == "" {
		return fmt.Errorf("influxdb: auth token is required")
	}
	if err := i.Tls.validate(); err != nil {
		return err
	}
	return nil
}

func (o *OfflineConfig) validate(c *CacheConfig, envId string) error {
	if !o.Enabled {
		return nil
	}
	if o.Local.FilePath == "" && !o.UseCache {
		return fmt.Errorf("sdk-" + envId + ": offline mode requires either a configured cache or a local file")
	}
	if o.Local.FilePath != "" && o.UseCache {
		return fmt.Errorf("sdk-" + envId + ": can't use both local file and cache for offline mode")
	}
	if o.Local.FilePath != "" {
		if err := o.Local.validate(envId); err != nil {
			return err
		}
	}
	if o.UseCache && !c.Redis.Enabled {
		return fmt.Errorf("sdk-" + envId + ": offline mode enabled with cache, but no cache is configured")
	}
	if o.UseCache && o.CachePollInterval < 1 {
		return fmt.Errorf("sdk-" + envId + ": cache poll interval must be greater than 1 seconds")
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
	if (w.Auth.User != "" && w.Auth.Password == "") || (w.Auth.Password != "" && w.Auth.User == "") {
		return fmt.Errorf("webhook: both basic auth user and password required")
	}
	return nil
}

func (l *LocalConfig) validate(envId string) error {
	if _, err := os.Stat(l.FilePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("sdk-"+envId+": couldn't find the local file %s", l.FilePath)
	}
	if l.Polling && l.PollInterval < 1 {
		return fmt.Errorf("sdk-" + envId + ": local file poll interval must be greater than 1 seconds")
	}
	return nil
}
