# goweb

[![Build Status](https://travis-ci.org/ViBiOh/goweb.svg?branch=master)](https://travis-ci.org/ViBiOh/goweb)
[![codecov](https://codecov.io/gh/ViBiOh/goweb/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/goweb)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/goweb)](https://goreportcard.com/report/github.com/ViBiOh/goweb)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/goweb)](https://dependabot.com)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_goweb&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_goweb)

## CI

Following variables are required for CI:

| Name | Purpose |
|:--:|:--:|
| **DOMAIN** | for setting Traefik domain for app |
| **DEPLOY_URL** | for deploying app to server |
| **DEPLOY_CREDENTIALS** | for deploying app to server |
| **DOCKER_USER** | for publishing Docker image |
| **DOCKER_PASS** | for publishing Docker image |

## Usage

```bash
Usage of api:
  -address string
        [http] Listen address {API_ADDRESS}
  -cert string
        [http] Certificate file {API_CERT}
  -corsCredentials
        [cors] Access-Control-Allow-Credentials {API_CORS_CREDENTIALS}
  -corsExpose string
        [cors] Access-Control-Expose-Headers {API_CORS_EXPOSE}
  -corsHeaders string
        [cors] Access-Control-Allow-Headers {API_CORS_HEADERS} (default "Content-Type")
  -corsMethods string
        [cors] Access-Control-Allow-Methods {API_CORS_METHODS} (default "GET")
  -corsOrigin string
        [cors] Access-Control-Allow-Origin {API_CORS_ORIGIN} (default "*")
  -csp string
        [owasp] Content-Security-Policy {API_CSP} (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options {API_FRAME_OPTIONS} (default "deny")
  -hsts
        [owasp] Indicate Strict Transport Security {API_HSTS} (default true)
  -key string
        [http] Key file {API_KEY}
  -location string
        [hello] TimeZone for displaying current time {API_LOCATION} (default "Europe/Paris")
  -okStatus int
        [http] Healthy HTTP Status code {API_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {API_PORT} (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics {API_PROMETHEUS_PATH} (default "/metrics")
  -swaggerTitle string
        [swagger] API Title {API_SWAGGER_TITLE} (default "API")
  -swaggerVersion string
        [swagger] API Version {API_SWAGGER_VERSION} (default "1.0.0")
  -url string
        [alcotest] URL to check {API_URL}
  -userAgent string
        [alcotest] User-Agent for check {API_USER_AGENT} (default "Alcotest")
```
