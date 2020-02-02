# vith

[![Build Status](https://travis-ci.org/ViBiOh/vith.svg?branch=master)](https://travis-ci.org/ViBiOh/vith)
[![codecov](https://codecov.io/gh/ViBiOh/vith/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/vith)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/vith)](https://goreportcard.com/report/github.com/ViBiOh/vith)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/vith)](https://dependabot.com)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_vith&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_vith)

## CI

Following variables are required for CI:

| Name | Purpose |
|:--:|:--:|
| **DOCKER_USER** | for publishing Docker image |
| **DOCKER_PASS** | for publishing Docker image |

## Usage

```bash
Usage of vith:
  -address string
        [http] Listen address {VITH_ADDRESS}
  -cert string
        [http] Certificate file {VITH_CERT}
  -key string
        [http] Key file {VITH_KEY}
  -okStatus int
        [http] Healthy HTTP Status code {VITH_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {VITH_PORT} (default 1080)
  -url string
        [alcotest] URL to check {VITH_URL}
  -userAgent string
        [alcotest] User-Agent for check {VITH_USER_AGENT} (default "Alcotest")
```
