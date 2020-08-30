# vith

[![Build Status](https://travis-ci.com/ViBiOh/vith.svg?branch=master)](https://travis-ci.com/ViBiOh/vith)
[![codecov](https://codecov.io/gh/ViBiOh/vith/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/vith)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/vith)](https://goreportcard.com/report/github.com/ViBiOh/vith)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_vith&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_vith)

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
        [http] Grace duration when SIGTERM received {VITH_GRACE_DURATION} (default "15s")
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
        [logger] Key for timestam in JSON {VITH_LOGGER_TIME_KEY} (default "time")
  -okStatus int
        [http] Healthy HTTP Status code {VITH_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {VITH_PORT} (default 1080)
  -url string
        [alcotest] URL to check {VITH_URL}
  -userAgent string
        [alcotest] User-Agent for check {VITH_USER_AGENT} (default "Alcotest")
```
