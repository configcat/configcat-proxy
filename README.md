# Configcat Proxy
https://configcat.com

ConfigCat is a feature flag and configuration management service that lets you separate releases from deployments. You can turn your features ON/OFF using <a href="https://app.configcat.com" target="_blank">ConfigCat Dashboard</a> even after they are deployed. ConfigCat lets you target specific groups of users based on region, email or any other custom user attribute.

ConfigCat is a <a href="https://configcat.com" target="_blank">hosted feature flag service</a>. Manage feature toggles across frontend, backend, mobile, desktop apps. <a href="https://configcat.com" target="_blank">Alternative to LaunchDarkly</a>. Management app + feature flag SDKs.

The ConfigCat Proxy provides a secure layer between your frontend, mobile or desktop applications and ConfigCat.

## Getting Started

### Installation
1. Pull docker image
```bash
docker pull configcat/configcat-proxy
```
2. Run the docker image
```bash
docker run \
   -e CONFIGCAT_SDK_KEY=##YOURSDKKEY## \
   -p 8081:8081 \
   configcat/configcat-proxy
```

### Usage
