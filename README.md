# vith

[![Build](https://github.com/ViBiOh/vith/workflows/Build/badge.svg)](https://github.com/ViBiOh/vith/actions)

## API

The HTTP API is pretty simple :

- `GET /health`: healthcheck of server, always respond [`okStatus (default 204)`](#usage)
- `GET /ready`: checks external dependencies availability and then respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when close signal is received
- `GET /version`: value of `VERSION` environment variable
- `POST /`: generate thumbnail of the video passed in payload in binary

### Installation

Golang binary is built with static link. You can download it directly from the [GitHub Release page](https://github.com/ViBiOh/vith/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/vith](https://hub.docker.com/r/vibioh/vith/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app)

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
  --address                     string    [server] Listen address ${VITH_ADDRESS}
  --amqpPrefetch                int       [amqp] Prefetch count for QoS ${VITH_AMQP_PREFETCH} (default 1)
  --amqpURI                     string    [amqp] Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost> ${VITH_AMQP_URI}
  --cert                        string    [server] Certificate file ${VITH_CERT}
  --exchange                    string    [thumbnail] AMQP Exchange Name ${VITH_EXCHANGE} (default "fibr")
  --graceDuration               duration  [http] Grace duration when signal received ${VITH_GRACE_DURATION} (default 30s)
  --idleTimeout                 duration  [server] Idle Timeout ${VITH_IDLE_TIMEOUT} (default 2m0s)
  --key                         string    [server] Key file ${VITH_KEY}
  --loggerJson                            [logger] Log format as JSON ${VITH_LOGGER_JSON} (default false)
  --loggerLevel                 string    [logger] Logger level ${VITH_LOGGER_LEVEL} (default "INFO")
  --loggerLevelKey              string    [logger] Key for level in JSON ${VITH_LOGGER_LEVEL_KEY} (default "level")
  --loggerMessageKey            string    [logger] Key for message in JSON ${VITH_LOGGER_MESSAGE_KEY} (default "msg")
  --loggerTimeKey               string    [logger] Key for timestamp in JSON ${VITH_LOGGER_TIME_KEY} (default "time")
  --name                        string    [server] Name ${VITH_NAME} (default "http")
  --okStatus                    int       [http] Healthy HTTP Status code ${VITH_OK_STATUS} (default 204)
  --port                        uint      [server] Listen port (0 to disable) ${VITH_PORT} (default 1080)
  --pprofAgent                  string    [pprof] URL of the Datadog Trace Agent (e.g. http://datadog.observability:8126) ${VITH_PPROF_AGENT}
  --pprofPort                   int       [pprof] Port of the HTTP server (0 to disable) ${VITH_PPROF_PORT} (default 0)
  --readTimeout                 duration  [server] Read Timeout ${VITH_READ_TIMEOUT} (default 2m0s)
  --routingKey                  string    [thumbnail] AMQP Routing Key to fibr ${VITH_ROUTING_KEY} (default "thumbnail_output")
  --shutdownTimeout             duration  [server] Shutdown Timeout ${VITH_SHUTDOWN_TIMEOUT} (default 10s)
  --storageFileSystemDirectory  /data     [storage] Path to directory. Default is dynamic. /data on a server and Current Working Directory in a terminal. ${VITH_STORAGE_FILE_SYSTEM_DIRECTORY}
  --storageObjectAccessKey      string    [storage] Storage Object Access Key ${VITH_STORAGE_OBJECT_ACCESS_KEY}
  --storageObjectBucket         string    [storage] Storage Object Bucket ${VITH_STORAGE_OBJECT_BUCKET}
  --storageObjectClass          string    [storage] Storage Object Class ${VITH_STORAGE_OBJECT_CLASS}
  --storageObjectEndpoint       string    [storage] Storage Object endpoint ${VITH_STORAGE_OBJECT_ENDPOINT}
  --storageObjectRegion         string    [storage] Storage Object Region ${VITH_STORAGE_OBJECT_REGION}
  --storageObjectSSL                      [storage] Use SSL ${VITH_STORAGE_OBJECT_SSL} (default true)
  --storageObjectSecretAccess   string    [storage] Storage Object Secret Access ${VITH_STORAGE_OBJECT_SECRET_ACCESS}
  --storagePartSize             uint      [storage] PartSize configuration ${VITH_STORAGE_PART_SIZE} (default 5242880)
  --streamExchange              string    [stream] Exchange name ${VITH_STREAM_EXCHANGE} (default "fibr")
  --streamExclusive                       [stream] Queue exclusive mode (for fanout exchange) ${VITH_STREAM_EXCLUSIVE} (default false)
  --streamInactiveTimeout       duration  [stream] When inactive during the given timeout, stop listening ${VITH_STREAM_INACTIVE_TIMEOUT} (default 0s)
  --streamMaxRetry              uint      [stream] Max send retries ${VITH_STREAM_MAX_RETRY} (default 3)
  --streamQueue                 string    [stream] Queue name ${VITH_STREAM_QUEUE} (default "stream")
  --streamRetryInterval         duration  [stream] Interval duration when send fails ${VITH_STREAM_RETRY_INTERVAL} (default 1h0m0s)
  --streamRoutingKey            string    [stream] RoutingKey name ${VITH_STREAM_ROUTING_KEY} (default "stream")
  --telemetryRate               string    [telemetry] OpenTelemetry sample rate, 'always', 'never' or a float value ${VITH_TELEMETRY_RATE} (default "always")
  --telemetryURL                string    [telemetry] OpenTelemetry gRPC endpoint (e.g. otel-exporter:4317) ${VITH_TELEMETRY_URL}
  --telemetryUint64                       [telemetry] Change OpenTelemetry Trace ID format to an unsigned int 64 ${VITH_TELEMETRY_UINT64} (default true)
  --thumbnailExchange           string    [thumbnail] Exchange name ${VITH_THUMBNAIL_EXCHANGE} (default "fibr")
  --thumbnailExclusive                    [thumbnail] Queue exclusive mode (for fanout exchange) ${VITH_THUMBNAIL_EXCLUSIVE} (default false)
  --thumbnailInactiveTimeout    duration  [thumbnail] When inactive during the given timeout, stop listening ${VITH_THUMBNAIL_INACTIVE_TIMEOUT} (default 0s)
  --thumbnailMaxRetry           uint      [thumbnail] Max send retries ${VITH_THUMBNAIL_MAX_RETRY} (default 3)
  --thumbnailQueue              string    [thumbnail] Queue name ${VITH_THUMBNAIL_QUEUE} (default "thumbnail")
  --thumbnailRetryInterval      duration  [thumbnail] Interval duration when send fails ${VITH_THUMBNAIL_RETRY_INTERVAL} (default 1h0m0s)
  --thumbnailRoutingKey         string    [thumbnail] RoutingKey name ${VITH_THUMBNAIL_ROUTING_KEY} (default "thumbnail")
  --tmpFolder                   string    [vith] Folder used for temporary files storage ${VITH_TMP_FOLDER} (default "/tmp")
  --url                         string    [alcotest] URL to check ${VITH_URL}
  --userAgent                   string    [alcotest] User-Agent for check ${VITH_USER_AGENT} (default "Alcotest")
  --writeTimeout                duration  [server] Write Timeout ${VITH_WRITE_TIMEOUT} (default 2m0s)
```
