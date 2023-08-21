package vith

import (
	"bytes"
	"flag"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	// SmallSize is the size of each thumbnail generated
	SmallSize = 150

	hlsExtension = ".m3u8"
)

var (
	bufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}

	slowClient = request.CreateClient(time.Minute, request.NoRedirection)
)

// App of package
type App struct {
	done               chan struct{}
	stop               chan struct{}
	streamRequestQueue chan model.Request
	storageApp         absto.Storage
	tracer             trace.Tracer
	amqpClient         *amqp.Client
	metric             metric.Int64Counter
	tmpFolder          string
	amqpExchange       string
	amqpRoutingKey     string
	imaginaryReq       request.Request
}

// Config of package
type Config struct {
	tmpFolder *string

	imaginaryURL  *string
	imaginaryUser *string
	imaginaryPass *string

	amqpExchange   *string
	amqpRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder: flags.New("TmpFolder", "Folder used for temporary files storage").Prefix(prefix).DocPrefix("vith").String(fs, "/tmp", overrides),

		imaginaryURL:  flags.New("ImaginaryURL", "Imaginary URL").Prefix(prefix).DocPrefix("thumbnail").String(fs, "http://image:9000", nil),
		imaginaryUser: flags.New("ImaginaryUser", "Imaginary Basic Auth User").Prefix(prefix).DocPrefix("thumbnail").String(fs, "", nil),
		imaginaryPass: flags.New("ImaginaryPassword", "Imaginary Basic Auth Password").Prefix(prefix).DocPrefix("thumbnail").String(fs, "", nil),

		amqpExchange:   flags.New("Exchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("thumbnail").String(fs, "fibr", overrides),
		amqpRoutingKey: flags.New("RoutingKey", "AMQP Routing Key to fibr").Prefix(prefix).DocPrefix("thumbnail").String(fs, "thumbnail_output", overrides),
	}
}

// New creates new App from Config
func New(config Config, amqpClient *amqp.Client, storageApp absto.Storage, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) App {
	imaginaryReq := request.Post(*config.imaginaryURL).WithClient(slowClient).BasicAuth(strings.TrimSpace(*config.imaginaryUser), *config.imaginaryPass)

	app := App{
		tmpFolder:  *config.tmpFolder,
		storageApp: storageApp,

		amqpClient:     amqpClient,
		amqpExchange:   strings.TrimSpace(*config.amqpExchange),
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),

		streamRequestQueue: make(chan model.Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
		imaginaryReq:       imaginaryReq,
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/vith/pkg/vith")

		var err error

		app.metric, err = meter.Int64Counter("vith.item")
		if err != nil {
			slog.Error("create vith counter", "err", err)
		}
	}

	if tracerProvider != nil {
		app.tracer = tracerProvider.Tracer("vith")
	}

	return app
}

// Handler for request. Should be use with net/http
func (a App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			a.handleHead(w, r)
		case http.MethodGet:
			a.handleGet(w, r)
		case http.MethodPost:
			a.handlePost(w, r)
		case http.MethodPut:
			a.handlePut(w, r)
		case http.MethodPatch:
			a.handlePatch(w, r)
		case http.MethodDelete:
			a.handleDelete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
