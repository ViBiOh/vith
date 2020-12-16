# vith

[![Build](https://github.com/ViBiOh/vith/workflows/Build/badge.svg)](https://github.com/ViBiOh/vith/actions)
[![codecov](https://codecov.io/gh/ViBiOh/vith/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/vith)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/vith)](https://goreportcard.com/report/github.com/ViBiOh/vith)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_vith&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_vith)

## API

The HTTP API is pretty simple :

- GET `/health` returns a status code (by default 204), if everything fine or HTTP/503 if not ready
- GET `/version` returns the value of the `VERSION` environment variable (by default the sha1 of the git commit), in plain text
- GET `/metrics` return the Prometheus metrics values
- POST `/` with the video in payload returns the thumbnail of the video, in binary

### Installation

Golang binary is built with static link. You can download it directly from the [Github Release page](https://github.com/ViBiOh/vith/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/vith](https://hub.docker.com/r/vibioh/vith/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/master/app)

## CI

Following variables are required for CI:

|      Name       |           Purpose           |
| :-------------: | :-------------------------: |
| **DOCKER_USER** | for publishing Docker image |
| **DOCKER_PASS** | for publishing Docker image |

## Usage

```bash
Usage of vith:
  -address string
        [http] Listen address {VITH_ADDRESS}
  -cert string
        [http] Certificate file {VITH_CERT}
  -graceDuration string
        [http] Grace duration when SIGTERM received {VITH_GRACE_DURATION} (default "30s")
  -idleTimeout string
        [http] Idle Timeout {VITH_IDLE_TIMEOUT} (default "2m")
  -key string
        [http] Key file {VITH_KEY}
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
        [http] Listen port {VITH_PORT} (default 1080)
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {VITH_PROMETHEUS_IGNORE}
  -prometheusPath string
        [prometheus] Path for exposing metrics {VITH_PROMETHEUS_PATH} (default "/metrics")
  -readTimeout string
        [http] Read Timeout {VITH_READ_TIMEOUT} (default "1m")
  -shutdownTimeout string
        [http] Shutdown Timeout {VITH_SHUTDOWN_TIMEOUT} (default "10s")
  -url string
        [alcotest] URL to check {VITH_URL}
  -userAgent string
        [alcotest] User-Agent for check {VITH_USER_AGENT} (default "Alcotest")
  -writeTimeout string
        [http] Write Timeout {VITH_WRITE_TIMEOUT} (default "1m")
```
