# ConfigCat Proxy Helm Chart

This Helm chart deploys the ConfigCat Proxy service on Kubernetes, with support for distributed cluster deployments and Redis caching.

## Overview

The ConfigCat Proxy provides a local caching proxy for ConfigCat feature flags and configurations. This chart can deploy ConfigCat Proxy in two modes:

1. **Standalone Mode**: A single ConfigCat Proxy deployment that connects directly to ConfigCat CDN
2. **Cluster Mode**: A distributed setup with online and offline proxies:
   - **Leader (ONLINE)**: Deployed in DMZ/external network with internet access to fetch configurations from ConfigCat CDN
   - **Followers (OFFLINE)**: Deployed in internal networks and get configurations from the shared Redis cache

## Installation

### Prerequisites
- Kubernetes 1.19+
- Helm 3.2.0+

### Installing the Chart

```bash
helm repo add configcat https://configcat.github.io/configcat-proxy
helm install configcat-proxy configcat/configcat-proxy \
  --set configcat.sdks.configurations='{\"production\":\"YOUR_PRODUCTION_SDK_KEY\",\"staging\":\"YOUR_STAGING_SDK_KEY\"}'
```

## SDK Configuration

The chart supports multiple ConfigCat environments (production, staging, development, etc.) within a single deployment. This allows you to configure different SDK keys for different environments and access them through the proxy using environment-specific endpoints.

### Accessing Different Environments

Once configured with multiple environments, you can access them via different SDK identifiers in your ConfigCat Proxy:

Example ConfigCat client configuration:
```yaml
# For production environment
configcat:
  sdk_key: "production"  # This maps to your production SDK key
  proxy_url: "http://configcat-proxy/configuration-files/production/config_v6.json"

# For staging environment  
configcat:
  sdk_key: "staging"     # This maps to your staging SDK key
  proxy_url: "http://configcat-proxy/configuration-files/staging/config_v6.json"
```

### Option 1: Direct Configuration (values.yaml)

You can specify SDK keys directly in your values:

```yaml
configcat:
  sdks:
    configurations:
      production: "your-production-sdk-key-123"
      staging: "your-staging-sdk-key-456"
      development: "your-dev-sdk-key-789"
```

### Option 2: Secret Reference (recommended for production)

For production deployments, it's recommended to store SDK keys in Kubernetes secrets:

1. First, create a Kubernetes secret containing your SDK configurations, i.e:

```bash
kubectl create secret generic configcat-sdks --from-literal=sdks-configuration='{"production":"your-production-sdk-key-123","staging":"your-staging-sdk-key-456","development":"your-dev-sdk-key-789"}'
```

2. Then reference this secret in your values:

```yaml
configcat:
  sdks:
    existingSecret: "configcat-sdks"
    existingSecretSdkConfigurationKey: "sdks-configuration"
```

This approach avoids storing sensitive SDK keys in your values files or Helm releases, enhancing security.

> **Note**: If both methods are provided, the secret reference takes precedence.

## Configuration Examples

### Standalone Mode (Default)

```yaml
# values.yaml
configcat:
  sdks:
    configurations:
      production: "YOUR_PRODUCTION_SDK_KEY"
      staging: "YOUR_STAGING_SDK_KEY"
      development: "YOUR_DEV_SDK_KEY"

# Redis configuration (optional)
redis:
  enabled: true
  auth:
    enabled: true
    # Option 1: Use existing secret (recommended)
    existingSecret: "redis-credentials"
    existingSecretPasswordKey: "redis-password"
    # Option 2: Set password directly (will be stored in secret automatically)
    password: "your-redis-password"
  database: 0
```

### Cluster Mode with Redis

```yaml
# values.yaml
configcat:
  sdks:
    # Reference SDK keys from a secret
    existingSecret: "configcat-sdks"
    existingSecretSdkConfigurationKey: "sdks-configuration"
    
# Enable cluster mode
clusterMode:
  enabled: true
  
  # Configure the online leader
  leader:
    enabled: true
    nodeSelector:
      network-zone: dmz
  
  # Configure offline followers
  followers:
    enabled: true
    replicaCount: 3
    nodeSelector:
      network-zone: internal

# Redis for shared caching
redis:
  enabled: true
  auth:
    enabled: true
    # Option 1: Use existing secret (recommended)
    existingSecret: "redis-credentials"
    existingSecretPasswordKey: "redis-password"
    # Option 2: Set password directly (chart creates secret automatically)
    password: "your-redis-password"
  # Use a specific database
  database: 1
```

## Redis Configuration

ConfigCat Proxy supports Redis configuration through native options rather than connection strings. This chart handles all the details and **only supports secret-based Redis authentication** for security.

### Redis Password Security

**Important**: Redis passwords are always stored in Kubernetes secrets, never as plain text in the configuration. You have two options:

1. **Existing Secret (Recommended)**: Reference a secret you create manually
2. **Direct Password**: The chart automatically creates a secret from the password value

### Redis Configuration Examples

```yaml
# Built-in Redis deployment
redis:
  enabled: true
  auth:
    enabled: true
    # Option 1: Use existing secret (recommended)
    existingSecret: "redis-credentials"
    existingSecretPasswordKey: "redis-password"
    # Option 2: Set password directly (chart creates secret automatically)
    password: "your-redis-password"
  database: 1  # Use Redis database 1

# External Redis (e.g., ElastiCache)
redis:
  enabled: false
externalRedis:
  host: "your-elasticache.example.com"  # Or array of hosts ["host1", "host2"]
  port: 6379
  database: 1
  auth:
    enabled: true
    existingSecret: "redis-credentials"
    existingSecretPasswordKey: "redis-password"

# Advanced Redis configuration
configcat:
  options:
    cache:
      type: "redis"
      redis:
        enabled: true
        # The following will be auto-populated from redis/externalRedis settings:
        # addresses, db, password
        # Advanced Redis settings:
        user: ""  # For Redis ACL
        tls:
          enabled: true
          min_version: "1.2"
          server_name: "redis.example.com"
          certificates:
            - cert: "/path/to/cert"
              key: "/path/to/key"
```

## Architecture

### Standalone Mode
```
┌─────────────┐     ┌───────────────┐
│  ConfigCat  │◄────┤  Single Proxy │◄────┐
│     CDN     │     └───────────────┘     │
└─────────────┘                           │
                                          │
                                    ┌─────────────┐
                                    │ Application │
                                    └─────────────┘
```

### Cluster Mode
```
┌─────────────┐     ┌───────────────────┐     ┌───────────────┐
│  ConfigCat  │◄────┤ Leader (ONLINE)   ├────►│     Redis     │
│     CDN     │     └───────────────────┘     └───────┬───────┘
└─────────────┘            in DMZ                     │
                                                      │
                                         ┌────────────▼──────────┐
                                         │ Followers (OFFLINE)   │◄─────┐
                                         └─────────────────────┘        │
                                          in Internal Network            │
                                                                   ┌─────────────┐
                                                                   │ Application │
                                                                   └─────────────┘
```

## Parameters

See the [values.yaml](values.yaml) file for the full list of parameters.

## License

This Helm chart is open-sourced software. Please refer to the ConfigCat licensing for more information.