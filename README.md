# ConfigCat Proxy [Beta]

[![Build Status](https://github.com/configcat/configcat-proxy/actions/workflows/proxy-ci.yml/badge.svg?branch=main)](https://github.com/configcat/configcat-proxy/actions/workflows/proxy-ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/configcat/configcat-proxy)](https://goreportcard.com/report/github.com/configcat/configcat-proxy)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=configcat_configcat-proxy&metric=alert_status)](https://sonarcloud.io/dashboard?id=configcat_configcat-proxy)

> [!NOTE]\
> The ConfigCat Proxy is in a public beta phase. If you have feedback or questions, please file a [GitHub Issue](https://github.com/configcat/configcat-proxy/issues) or [contact us](https://configcat.com/support).

The ConfigCat Proxy allows you to host a feature flag evaluation service in your own infrastructure. 
It's a small Go application that communicates with ConfigCat's CDN network and caches/proxies *config JSON* files for your frontend and backend applications. 
The *config JSON* contains all the data that is needed for ConfigCat SDKs to evaluate feature flags.

The ConfigCat Proxy provides the following:
- **Performance**: The Proxy can be deployed close to your applications and can serve the downloaded *config JSON* files from memory. ConfigCat SDKs then can operate on the [proxied *config JSON*](https://configcat.com/docs/advanced/proxy/endpoints#cdn-proxy). This can reduce the duration of feature flag evaluation for stateless or short-lived applications.
- **Reliability**: The Proxy can store the downloaded *config JSON* files in an external [cache](https://configcat.com/docs/advanced/proxy/proxy-overview#cache). It can fall back to operating on the cached *config JSON* if the ConfigCat CDN network becomes inaccessible.
- **Security**: The Proxy can act as a [server side flag evaluation](https://configcat.com/docs/advanced/proxy/endpoints#api) component. Using it like that can prevent the exposure of *config JSON* files to frontend and mobile applications.
- **Scalability**: Horizontal scaling allows you to align with the load coming from your applications accordingly.
- **Streaming**: The Proxy provides real-time feature flag change notifications via [Server-Sent Events (SSE)](https://configcat.com/docs/advanced/proxy/endpoints#sse) and [gRPC](https://configcat.com/docs/advanced/proxy/grpc).

To learn more, read the [documentation](https://configcat.com/docs/advanced/proxy/proxy-overview).

### How It Works
The Proxy wraps one or more SDK instances for handling feature flag evaluation requests. It also serves the related *config JSON* files that can be consumed by other ConfigCat SDKs running in your applications.

Within the Proxy, the underlying SDK instances can run in the following modes:
- **Online**: In this mode, the underlying SDK has an active connection to the ConfigCat CDN network through the internet.
- **Offline**: In [this mode](https://configcat.com/docs/advanced/proxy/proxy-overview#offline-mode), the underlying SDK doesn't have an active connection to ConfigCat. Instead, it uses the configured cache or a file as a source of its *config JSON*.

### Communication

There are three ways how the Proxy is informed about the availability of new feature flag evaluation data:
- **Polling**: The ConfigCat SDKs within the Proxy are regularly polling the ConfigCat CDN for new *config JSON* versions.
- **Webhook**: The Proxy has [webhook endpoints](https://configcat.com/docs/advanced/proxy/endpoints#webhook) (for each underlying SDK), which can be set on the <a target="_blank" href="https://app.configcat.com/product/webhooks">ConfigCat Dashboard</a> to be invoked when someone saves & publishes new feature flag changes.
- **Cache polling / file watching**: In [offline mode](https://configcat.com/docs/advanced/proxy/proxy-overview#offline-mode), the Proxy can regularly poll a cache or watch a file for new *config JSON* versions.

These are the ports used by the Proxy by default:
- **8050**: for standard HTTP communication. ([API](https://configcat.com/docs/advanced/proxy/endpoints#api), [CDN proxy](https://configcat.com/docs/advanced/proxy/endpoints#cdn-proxy), [Webhook](https://configcat.com/docs/advanced/proxy/endpoints#webhook), [SSE](https://configcat.com/docs/advanced/proxy/endpoints#sse))
- **8051**: for providing diagnostic data ([status](https://configcat.com/docs/advanced/proxy/monitoring#status), [prometheus metrics](https://configcat.com/docs/advanced/proxy/monitoring#prometheus-metrics).
- **50051**: for [gRPC](https://configcat.com/docs/advanced/proxy/grpc) communication.

## Installation

You can install the ConfigCat Proxy from the following sources:

### Docker

The docker image is available on DockerHub. You can run the image either as a standalone docker container or via `docker-compose`.

1. Pull the latest image:
    ```shell
    docker pull configcat/proxy:latest
    ```
2. Run the ConfigCat Proxy:
    ```shell
    docker run -d --name configcat-proxy \ 
      -p 8050:8050 -p 8051:8051 -p 50051:50051 \
      -e CONFIGCAT_SDKS='{"<sdk-identifier>":"<your-sdk-key>"}' \
      configcat/proxy
    ```

Using with `docker-compose`:

1. Put the following into your `docker-compose.yml`:
    ```yaml
    services:
      configcat_proxy:
        image: configcat/proxy:latest
        environment:
          - CONFIGCAT_SDKS={"<sdk-identifier>":"<your-sdk-key>"}
        ports:
          - "8050:8050"
          - "8051:8051"
          - "50051:50051"
    ```
2. Start docker services by executing the following command:
    ```shell
    docker-compose up -f docker-compose.yml -d
    ```

### Standalone Executables

You can download the executables directly from <a target="_blank" href="https://github.com/configcat/configcat-proxy/releases">GitHub Releases</a> for your desired platform.

## Health Check
After installation, you can check the [status endpoint](https://configcat.com/docs/advanced/proxy/monitoring#status) of the Proxy to ensure it's working correctly: `http://localhost:8051/status`

## Need help?
https://configcat.com/support

## Contributing
Contributions are welcome. Please read the [Contribution Guideline](CONTRIBUTING.md) for more info.

## About ConfigCat

ConfigCat is a feature flag, feature toggle, and configuration management service that lets you launch new features and change your software configuration remotely without actually (re)deploying code. ConfigCat even helps you do controlled roll-outs like canary releases and blue-green deployments.

ConfigCat is a [hosted feature flag service](https://configcat.com). Manage feature toggles across frontend, backend, mobile, and desktop apps. [Alternative to LaunchDarkly](https://configcat.com). Management app + feature flag SDKs.

- [Documentation](https://configcat.com/docs)
- [Blog](https://blog.configcat.com)
