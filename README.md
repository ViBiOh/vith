# vith

[![Build](https://github.com/ViBiOh/vith/workflows/Build/badge.svg)](https://github.com/ViBiOh/vith/actions)
[![codecov](https://codecov.io/gh/ViBiOh/vith/branch/main/graph/badge.svg)](https://codecov.io/gh/ViBiOh/vith)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_vith&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_vith)

## API

The HTTP API is pretty simple :

- `GET /health`: healthcheck of server, always respond [`okStatus (default 204)`](#usage)
- `GET /ready`: checks external dependencies availability and then respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when `SIGTERM` is received
- `GET /version`: value of `VERSION` environment variable
- `GET /metrics`: Prometheus metrics, on a dedicated port [`prometheusPort (default 9090)`](#usage)
- `POST /`: generate thumbnail of the video passed in payload in binary

### Installation

Golang binary is built with static link. You can download it directly from the [Github Release page](https://github.com/ViBiOh/vith/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/vith](https://hub.docker.com/r/vibioh/vith/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app)

## CI

Following variables are required for CI:

|      Name       |           Purpose           |
| :-------------: | :-------------------------: |
| **DOCKER_USER** | for publishing Docker image |
| **DOCKER_PASS** | for publishing Docker image |

## Usage

The application can be configured by passing CLI args described below or their equivalent as environment variable. CLI values take precedence over environments variables.

Be careful when using the CLI values, if someone list the processes on the system, they will appear in plain-text. Pass secrets by environment variables: it's less easily visible.

```bash
Usage of vith:
  -address string
        [server] Listen address {VITH_ADDRESS}
  -amqpExchange string
        [amqp] Exchange name {VITH_AMQP_EXCHANGE} (default "fibr")
  -amqpMaxRetry uint
        [amqp] Max send retries {VITH_AMQP_MAX_RETRY} (default 3)
  -amqpQueue string
        [amqp] Queue name {VITH_AMQP_QUEUE} (default "vith")
  -amqpRetryInterval string
        [amqp] Interval duration when send fails {VITH_AMQP_RETRY_INTERVAL} (default "1h")
  -amqpRoutingKey string
        [amqp] RoutingKey name {VITH_AMQP_ROUTING_KEY} (default "stream")
  -amqpURI string
        [amqp] Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost> {VITH_AMQP_URI}
  -cert string
        [server] Certificate file {VITH_CERT}
  -graceDuration string
        [http] Grace duration when SIGTERM received {VITH_GRACE_DURATION} (default "30s")
  -idleTimeout string
        [server] Idle Timeout {VITH_IDLE_TIMEOUT} (default "2m")
  -key string
        [server] Key file {VITH_KEY}
  -loggerJson
        [logger] Log format as JSON {VITH_LOGGER_JSON}
  -loggerLevel string
        [logger] Logger level {VITH_LOGGER_LEVEL} (default "INFO")
  -loggerLevelKey string
        [logger] Key for level in JSON {VITH_LOGGER_LEVEL_KEY} (default "level")
  -loggerMessageKey string
        [logger] Key for message in JSON {VITH_LOGGER_MESSAGE_KEY} (default "message")
  -loggerTimeKey string
        [logger] Key for timestamp in JSON {VITH_LOGGER_TIME_KEY} (default "time")
  -okStatus int
        [http] Healthy HTTP Status code {VITH_OK_STATUS} (default 204)
  -port uint
        [server] Listen port (0 to disable) {VITH_PORT} (default 1080)
  -prometheusAddress string
        [prometheus] Listen address {VITH_PROMETHEUS_ADDRESS}
  -prometheusCert string
        [prometheus] Certificate file {VITH_PROMETHEUS_CERT}
  -prometheusGzip
        [prometheus] Enable gzip compression of metrics output {VITH_PROMETHEUS_GZIP}
  -prometheusIdleTimeout string
        [prometheus] Idle Timeout {VITH_PROMETHEUS_IDLE_TIMEOUT} (default "10s")
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {VITH_PROMETHEUS_IGNORE}
  -prometheusKey string
        [prometheus] Key file {VITH_PROMETHEUS_KEY}
  -prometheusPort uint
        [prometheus] Listen port (0 to disable) {VITH_PROMETHEUS_PORT} (default 9090)
  -prometheusReadTimeout string
        [prometheus] Read Timeout {VITH_PROMETHEUS_READ_TIMEOUT} (default "5s")
  -prometheusShutdownTimeout string
        [prometheus] Shutdown Timeout {VITH_PROMETHEUS_SHUTDOWN_TIMEOUT} (default "5s")
  -prometheusWriteTimeout string
        [prometheus] Write Timeout {VITH_PROMETHEUS_WRITE_TIMEOUT} (default "10s")
  -readTimeout string
        [server] Read Timeout {VITH_READ_TIMEOUT} (default "2m")
  -shutdownTimeout string
        [server] Shutdown Timeout {VITH_SHUTDOWN_TIMEOUT} (default "10s")
  -tmpFolder string
        [vith] Folder used for temporary files storage {VITH_TMP_FOLDER} (default "/tmp")
  -url string
        [alcotest] URL to check {VITH_URL}
  -userAgent string
        [alcotest] User-Agent for check {VITH_USER_AGENT} (default "Alcotest")
  -workDir string
        [vith] Working directory for GET requests {VITH_WORK_DIR}
  -writeTimeout string
        [server] Write Timeout {VITH_WRITE_TIMEOUT} (default "2m")
```
