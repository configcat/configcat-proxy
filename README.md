# Configcat Proxy
https://configcat.com

ConfigCat is a feature flag and configuration management service that lets you separate releases from deployments. You can turn your features ON/OFF using <a href="https://app.configcat.com" target="_blank">ConfigCat Dashboard</a> even after they are deployed. ConfigCat lets you target specific groups of users based on region, email or any other custom user attribute.

ConfigCat is a <a href="https://configcat.com" target="_blank">hosted feature flag service</a>. Manage feature toggles across frontend, backend, mobile, desktop apps. <a href="https://configcat.com" target="_blank">Alternative to LaunchDarkly</a>. Management app + feature flag SDKs.

The ConfigCat Proxy provides a secure layer between your frontend, mobile or desktop applications and ConfigCat. The ConfigCat Proxy is a simple NodeJs application that uses the [https://configcat.com/docs/sdk-reference/node](ConfigCat Node.js SDK) to serve feature flag values.

[![ConfigCat Proxy CI](https://github.com/configcat/configcat-proxy/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/configcat/configcat-proxy/actions/workflows/ci.yml)
![License](https://img.shields.io/github/license/configcat/configcat-proxy.svg)

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
   -p 4224:4224
   configcat/configcat-proxy
```
3. Test
```bash
curl http://localhost:4224/health  -I
```

### Configuration
Configuration is available through environment variables.
| Name | Description | Default value |
| --------- | ----------- | ----------- |
| `CONFIGCAT_SDK_KEY` | SDK Key to access your feature flags and configurations. Get it from ConfigCat Dashboard. | *required* |
| `CONFIGCAT_DATA_GOVERNANCE` | Describes the location of your feature flag and setting data within the ConfigCat CDN. This parameter needs to be in sync with your Data Governance preferences. Read more at https://configcat.com/docs/advanced/data-governance. Available options: `Global`, `EuOnly`. | `Global` |
| `CONFIGCAT_POLLING_MODE` | The ConfigCat SDK supports 3 different polling mechanisms to acquire the setting values from ConfigCat. Read more at https://configcat.com/docs/sdk-reference/node/#polling-modes. Available options: `AutoPoll`, `LazyLoad`, `ManualPoll`. | `AutoPoll` |
| `CONFIGCAT_REQUEST_TIMEOUT_MS` | The amount of milliseconds the SDK waits for a response from the ConfigCat servers before returning values from the cache. | `30000` |
| `CONFIGCAT_AUTOPOLL_POLL_INTERVAL_SECONDS` | Polling interval. Range: 1 - Number.MAX_SAFE_INTEGER. Only available for the `AutoPoll` mode. | `60` |
| `CONFIGCAT_AUTOPOLL_MAX_INIT_WAIT_TIME_SECONDS` | Maximum waiting time between the client initialization and the first config acquisition in seconds. Only available for the `AutoPoll` mode. | `5` |
| `CONFIGCAT_LAZYLOAD_CACHE_TIME_TO_LIVE_SECONDS` | Maximum waiting time between the client initialization and the first config acquisition in seconds. Only available for the `LazyLoad` mode. | `60` |

## Usage
Avaiable endpoints:

### `getValue`
Method: `POST`  
Evaluates a setting by the parameters and returns the value.
```bash
curl -X POST http://localhost:4224/getValue \
  -H 'Content-Type: application/json' \
  -d '##BODY##'
```

#### Request Body
##### Easy way
Request Body:
```json
{
    "key": "isMyAwesomeFeatureEnabled",
    "defaultValue": false
}
```

##### Simple user targeting
Request Body:
```json
{
    "key": "isMyAwesomeFeatureEnabled",
    "defaultValue": false,
    "user":
    {
        "identifier": "##SOME-USER-IDENTIFIER##"
    }
}
```

##### Advanced user targeting
Request Body:
```json
{
    "key": "isMyAwesomeFeatureEnabled",
    "defaultValue": false,
    "user":
    {
        "identifier": "##SOME-USER-IDENTIFIER##",
        "email": "jane@example.com",
        "country": "Awesomnia",
        "custom":
        {
            "SubscriptionType": "Pro",
            "UserRole": "Knight of Awesomnia"
        }
    }
}
```

#### Response
```json
{
    "value": false
}
```

### `getAllValues`
Method: `POST`  
Evaluates all of the settings by the parameters and returns the values.
```bash
curl -X POST http://localhost:4224/getAllValues \
  -H 'Content-Type: application/json' \
  -d '##BODY##'
```

#### Request Body
##### Easy way
Request Body:
```json
{}
```

##### Simple user targeting
Request Body:
```json
{
    "user":
    {
        "identifier": "##SOME-USER-IDENTIFIER##"
    }
}
```

##### Advanced user targeting
Request Body:
```json
{
    "user":
    {
        "identifier": "##SOME-USER-IDENTIFIER##",
        "email": "jane@example.com",
        "country": "Awesomnia",
        "custom":
        {
            "SubscriptionType": "Pro",
            "UserRole": "Knight of Awesomnia"
        }
    }
}
```

#### Response
```json
[
    {
        "settingKey": "isMyAwesomeFeatureEnabled",
        "settingValue": false
    },
    {
        "settingKey": "isMySecondAwesomeFeatureEnabled",
        "settingValue": false
    }
]
```

### `getAllKeys`
Method: `POST`  
Returns all of the setting keys.

```bash
curl -X POST http://localhost:4224/getAllValues \
  -H 'Content-Type: application/json' \
  -d '##BODY##'
```
```json
[
    "isMyAwesomeFeatureEnabled",
    "isMySecondAwesomeFeatureEnabled"
]
```
#### Response

### `forceRefresh`
Method: `POST`  
Refreshes the ConfigCatClient cache. Especially useful for `Manual Poll` mode combined with (https://configcat.com/docs/advanced/notifications-webhooks)[ConfigCat Webhooks].
```bash
curl -X POST http://localhost:4224/forceRefresh
```

### `health`
Method: `GET`
Returns a success response if the service is up and running.
```bash
curl http://localhost:4224/health  -I
```

## Need help?
https://configcat.com/support

## Contributing
Contributions are welcome. For more info please read the [Contribution Guideline](CONTRIBUTING.md).

## About ConfigCat
- [Documentation](https://configcat.com/docs)
- [Blog](https://configcat.com/blog)