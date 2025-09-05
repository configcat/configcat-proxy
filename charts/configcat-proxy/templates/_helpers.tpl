{{/*
Expand the name of the chart.
*/}}
{{- define "configcat-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "configcat-proxy.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create leader name
*/}}
{{- define "configcat-proxy.leaderName" -}}
{{ include "configcat-proxy.fullname" . }}-leader
{{- end }}

{{/*
Create follower name
*/}}
{{- define "configcat-proxy.followerName" -}}
{{ include "configcat-proxy.fullname" . }}-follower
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "configcat-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Base labels without component - used for base resources
*/}}
{{- define "configcat-proxy.baseLabels" -}}
helm.sh/chart: {{ include "configcat-proxy.chart" . }}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Leader labels
*/}}
{{- define "configcat-proxy.leaderLabels" -}}
helm.sh/chart: {{ include "configcat-proxy.chart" . }}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: leader
{{- end }}

{{/*
Leader selector labels - explicitly for selectors only
*/}}
{{- define "configcat-proxy.leaderSelectorLabels" -}}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: leader
{{- end }}

{{/*
Follower labels
*/}}
{{- define "configcat-proxy.followerLabels" -}}
helm.sh/chart: {{ include "configcat-proxy.chart" . }}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: follower
{{- end }}

{{/*
Follower selector labels - explicitly for selectors only
*/}}
{{- define "configcat-proxy.followerSelectorLabels" -}}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: follower
{{- end }}

{{/*
Common labels - for resources that don't need component
*/}}
{{- define "configcat-proxy.labels" -}}
helm.sh/chart: {{ include "configcat-proxy.chart" . }}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels - base selector without component
*/}}
{{- define "configcat-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "configcat-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "configcat-proxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "configcat-proxy.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return Redis addresses array for internal Redis
This helper is kept for compatibility but is replaced with direct address handling
*/}}
{{- define "configcat-proxy.redisAddresses" -}}
{{- if .Values.redis.enabled -}}
["{{ .Release.Name }}-redis-master.{{ .Release.Namespace }}.svc.cluster.local:6379"]
{{- else if and (not .Values.redis.enabled) .Values.externalRedis.host -}}
{{- if kindIs "string" .Values.externalRedis.host -}}
["{{ .Values.externalRedis.host }}:{{ .Values.externalRedis.port }}"]
{{- else -}}
{{ .Values.externalRedis.host | toJson }}
{{- end -}}
{{- else -}}
[]
{{- end -}}
{{- end }}

{{/*
Generate options.yml content for leader node
*/}}
{{- define "configcat-proxy.optionsYamlTpl" -}}
{{- if .Values.configcat.sdks.existingSecret }}
{{/* When using secrets, generate only the SDKs that will be replaced */}}
{{/* The init container will dynamically replace these placeholders based on what's in the secret */}}
sdks:
{{- else }}
{{/* Direct configuration from values */}}
sdks:
{{- range $envName, $sdkKey := .Values.configcat.sdks.configurations }}
  {{ $envName }}:
    key: "__SDK_KEY_{{ upper $envName }}__"
    base_url: "{{ $.Values.configcat.options.baseUrl }}"
    poll_interval: {{ $.Values.configcat.options.pollIntervalSeconds }}
    data_governance: "{{ lower $.Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ $.Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ $.Values.configcat.options.logLevel }}"
      enabled: false
      use_cache: true
      cache_poll_interval: {{ $.Values.configcat.options.followers.cachePollInterval }}
{{- end }}
{{- end }}
cache:
{{- if or .Values.redis.enabled .Values.externalRedis.host }}
  redis:
    enabled: true
    db: {{ .Values.externalRedis.database | default 0 }}
    addresses: 
    {{- if .Values.redis.enabled }}
    - "{{ .Release.Name }}-redis-master.{{ .Release.Namespace }}.svc.cluster.local:6379"
    {{- else if .Values.externalRedis.host }}
    - "{{ .Values.externalRedis.host }}:{{ .Values.externalRedis.port }}"
    {{- end }}
    password: "__REDIS_PASSWORD__"
    tls:
      enabled: {{ .Values.configcat.options.cache.redis.tls.enabled }}
      {{- if .Values.configcat.options.cache.redis.tls.min_version }}
      min_version: {{ .Values.configcat.options.cache.redis.tls.min_version }}
      {{- end }}
{{- else }}
  # In-memory cache (no external dependencies)
  # ConfigCat Proxy will use internal memory storage when no external cache is configured
{{- end }}
{{- end -}}

{{/*
Generate options.yml for the follower node 
*/}}
{{- define "configcat-proxy.followerOptionsYamlTpl" -}}
{{- if .Values.configcat.sdks.existingSecret }}
{{/* When using secrets, generate only the SDKs that will be replaced */}}
{{/* The init container will dynamically generate these based on what's in the secret */}}
sdks:
{{- else }}
{{/* Direct configuration from values */}}
sdks:
{{- range $envName, $sdkKey := .Values.configcat.sdks.configurations }}
  {{ $envName }}:
    key: "__SDK_KEY_{{ upper $envName }}__"
    data_governance: "{{ lower $.Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ $.Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ $.Values.configcat.options.logLevel }}"
      enabled: true
      use_cache: true
      cache_poll_interval: {{ $.Values.configcat.options.followers.cachePollInterval }}
{{- end }}
{{- end }}
cache:
  redis:
    enabled: true
    db: {{ .Values.externalRedis.database | default 0 }}
    addresses: 
    {{- if .Values.redis.enabled }}
    - "{{ .Release.Name }}-redis-master.{{ .Release.Namespace }}.svc.cluster.local:6379"
    {{- else if .Values.externalRedis.host }}
    - "{{ .Values.externalRedis.host }}:{{ .Values.externalRedis.port }}"
    {{- end }}
    password: "__REDIS_PASSWORD__"
    tls:
      enabled: {{ .Values.configcat.options.cache.redis.tls.enabled }}
      {{- if .Values.configcat.options.cache.redis.tls.min_version }}
      min_version: {{ .Values.configcat.options.cache.redis.tls.min_version }}
      {{- end }}
{{- end -}}

{{/*
Shared init container script for configuration preparation
*/}}
{{- define "configcat-proxy.initContainerScript" -}}
set -e

# Create the target directory
mkdir -p /config-dir

# Read the template file
TEMPLATE=$(cat /template/options.yml.tpl)
if [ -z "$TEMPLATE" ]; then
  echo "ERROR: Template file is empty or not found"
  exit 1
fi

# Start with the template
RESULT="$TEMPLATE"

{{- if .Values.configcat.sdks.existingSecret }}
# Parse SDK configurations from JSON and generate dynamic configuration
if [ -n "$CONFIGCAT_SDK_KEY" ]; then
  # Use DEPLOYMENT_TYPE environment variable to determine configuration type
  if [ "$DEPLOYMENT_TYPE" = "follower" ]; then
    OFFLINE_MODE="true"
    echo "Detected follower configuration (offline mode)"
  else
    OFFLINE_MODE="false"
    echo "Detected leader configuration (online mode)"
  fi
  
  if command -v jq >/dev/null 2>&1; then
    # Use jq to dynamically build SDK configurations
    echo "$CONFIGCAT_SDK_KEY" | jq -r 'keys[]' | while read env_name; do
      sdk_key=$(echo "$CONFIGCAT_SDK_KEY" | jq -r ".[\"$env_name\"]")
      if [ "$OFFLINE_MODE" = "true" ]; then
        # Follower configuration (offline mode)
        cat >> /tmp/sdk_configs.yml << EOF
  $env_name:
    key: "$sdk_key"
    data_governance: "{{ lower .Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ .Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ .Values.configcat.options.logLevel }}"
      enabled: true
      use_cache: true
      cache_poll_interval: {{ .Values.configcat.options.followers.cachePollInterval }}
EOF
      else
        # Leader configuration (online mode)
        cat >> /tmp/sdk_configs.yml << EOF
  $env_name:
    key: "$sdk_key"
    base_url: "{{ .Values.configcat.options.baseUrl }}"
    poll_interval: {{ .Values.configcat.options.pollIntervalSeconds }}
    data_governance: "{{ lower .Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ .Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ .Values.configcat.options.logLevel }}"
      enabled: false
      use_cache: true
      cache_poll_interval: {{ .Values.configcat.options.followers.cachePollInterval }}
EOF
      fi
      echo "Added SDK configuration for environment: $env_name (offline: $OFFLINE_MODE)"
    done
    
    # Replace the sdks: placeholder with actual configurations
    if [ -f /tmp/sdk_configs.yml ]; then
      RESULT=$(echo "$RESULT" | sed "/^sdks:$/r /tmp/sdk_configs.yml")
      rm -f /tmp/sdk_configs.yml
    fi
  else
    # Fallback: simple approach for environments we find
    echo "jq not available, using fallback approach"
    for env_name in production staging development dev test default; do
      SDK_KEY=$(echo "$CONFIGCAT_SDK_KEY" | sed -n "s/.*\"$env_name\":\"\\([^\"]*\\)\".*/\\1/p")
      if [ -n "$SDK_KEY" ]; then
        if [ "$OFFLINE_MODE" = "true" ]; then
          # Follower configuration
          cat >> /tmp/sdk_configs.yml << EOF
  $env_name:
    key: "$SDK_KEY"
    data_governance: "{{ lower .Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ .Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ .Values.configcat.options.logLevel }}"
      enabled: true
      use_cache: true
      cache_poll_interval: {{ .Values.configcat.options.followers.cachePollInterval }}
EOF
        else
          # Leader configuration
          cat >> /tmp/sdk_configs.yml << EOF
  $env_name:
    key: "$SDK_KEY"
    base_url: "{{ .Values.configcat.options.baseUrl }}"
    poll_interval: {{ .Values.configcat.options.pollIntervalSeconds }}
    data_governance: "{{ lower .Values.configcat.options.dataGovernance }}"
    log:
      level: "{{ .Values.configcat.options.logLevel }}"
    offline:
      log:
        level: "{{ .Values.configcat.options.logLevel }}"
      enabled: false
      use_cache: true
      cache_poll_interval: {{ .Values.configcat.options.followers.cachePollInterval }}
EOF
        fi
        echo "Added SDK configuration for environment: $env_name (offline: $OFFLINE_MODE)"
      fi
    done
    
    # Replace the sdks: placeholder with actual configurations
    if [ -f /tmp/sdk_configs.yml ]; then
      RESULT=$(echo "$RESULT" | sed "/^sdks:$/r /tmp/sdk_configs.yml")
      rm -f /tmp/sdk_configs.yml
    fi
  fi
else
  echo "WARNING: CONFIGCAT_SDK_KEY environment variable is empty"
fi
{{- else }}
# Direct configuration: Replace placeholders with values from template
{{- range $envName, $sdkKey := .Values.configcat.sdks.configurations }}
RESULT=$(echo "$RESULT" | sed "s|__SDK_KEY_{{ upper $envName }}__|{{ $sdkKey }}|g")
echo "Replaced placeholder for environment: {{ $envName }}"
{{- end }}
{{- end }}

# Handle Redis password replacement
{{- if and .Values.externalRedis.auth.enabled .Values.externalRedis.auth.existingSecret }}
if [ -n "$CONFIGCAT_CACHE_REDIS_PASSWORD" ]; then
  FINAL_CONFIG=$(echo "$RESULT" | sed "s|__REDIS_PASSWORD__|$CONFIGCAT_CACHE_REDIS_PASSWORD|g")
  echo "Redis password configured from external secret"
else
  echo "WARNING: Redis password is empty"
  FINAL_CONFIG=$(echo "$RESULT" | sed "s|__REDIS_PASSWORD__||g")
fi
{{- else if .Values.redis.enabled }}
if [ -n "$CONFIGCAT_CACHE_REDIS_PASSWORD" ]; then
  FINAL_CONFIG=$(echo "$RESULT" | sed "s|__REDIS_PASSWORD__|$CONFIGCAT_CACHE_REDIS_PASSWORD|g")
  echo "Redis password configured from internal Redis"
else
  echo "WARNING: Redis password is empty"
  FINAL_CONFIG=$(echo "$RESULT" | sed "s|__REDIS_PASSWORD__||g")
fi
{{- else }}
FINAL_CONFIG=$(echo "$RESULT" | sed "s|__REDIS_PASSWORD__||g")
echo "Redis not enabled, removing password placeholder"
{{- end }}

# Apply Redis connection settings
{{- if .Values.redis.enabled }}
FINAL_CONFIG=$(echo "$FINAL_CONFIG" | sed "s|__REDIS_TIMEOUT__|{{ .Values.redis.connection.timeout | default "5s" }}|g")
FINAL_CONFIG=$(echo "$FINAL_CONFIG" | sed "s|__REDIS_RETRIES__|{{ .Values.redis.connection.retries | default "3" }}|g")
echo "Redis connection settings applied"
{{- end }}

# Write final configuration
echo "$FINAL_CONFIG" > /config-dir/options.yml

# Validate configuration file was created
if [ ! -f /config-dir/options.yml ]; then
  echo "ERROR: Configuration file was not created"
  exit 1
fi

# Log success with file size for verification
CONFIG_SIZE=$(wc -c < /config-dir/options.yml)
echo "Configuration prepared successfully at /config-dir/options.yml (${CONFIG_SIZE} bytes)"
{{- end -}}

{{/*
Shared environment variables for init container
*/}}
{{- define "configcat-proxy.initContainerEnv" -}}
{{- if .Values.configcat.sdks.existingSecret }}
- name: CONFIGCAT_SDK_KEY
  valueFrom:
    secretKeyRef:
      name: {{.Values.configcat.sdks.existingSecret}}
      key: {{.Values.configcat.sdks.existingSecretSdkConfigurationKey | default "sdk-key"}}
{{- end }}
- name: DEPLOYMENT_TYPE
  value: "{{ .deploymentType | default "leader" }}"
{{- if and .Values.externalRedis.auth.enabled .Values.externalRedis.auth.existingSecret }}
- name: CONFIGCAT_CACHE_REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{.Values.externalRedis.auth.existingSecret}}
      key: {{.Values.externalRedis.auth.existingSecretPasswordKey}}
{{- else if .Values.redis.enabled }}
- name: CONFIGCAT_CACHE_REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{.Release.Name}}-redis
      key: redis-password
{{- end }}
{{- end -}}

{{/*
Complete init container definition for reuse across deployments
*/}}
{{- define "configcat-proxy.initContainer" -}}
- name: config-init
  image: busybox
  command:
    - /bin/sh
    - -c
    - |
{{ include "configcat-proxy.initContainerScript" . | indent 6 }}
  env:
{{ include "configcat-proxy.initContainerEnv" . | indent 4 }}
  volumeMounts:
    - name: template-volume
      mountPath: /template
    - name: config-volume
      mountPath: /config-dir
{{- end -}}
