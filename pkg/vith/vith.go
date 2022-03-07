package vith

import (
	"bytes"
	"flag"
	"net/http"
	"strings"
	"sync"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/flags"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/vith/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	// SmallSize is the size of each thumbnail generated
	SmallSize = 150

	hlsExtension = ".m3u8"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
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
	metric             *prometheus.CounterVec
	tmpFolder          string
	imaginaryReq       request.Request
}

// Config of package
type Config struct {
	tmpFolder *string

	imaginaryURL  *string
	imaginaryUser *string
	imaginaryPass *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder: flags.New(prefix, "vith", "TmpFolder").Default("/tmp", overrides).Label("Folder used for temporary files storage").ToString(fs),

		imaginaryURL:  flags.New(prefix, "thumbnail", "ImaginaryURL").Default("http://image:9000", nil).Label("Imaginary URL").ToString(fs),
		imaginaryUser: flags.New(prefix, "thumbnail", "ImaginaryUser").Default("", nil).Label("Imaginary Basic Auth User").ToString(fs),
		imaginaryPass: flags.New(prefix, "thumbnail", "ImaginaryPassword").Default("", nil).Label("Imaginary Basic Auth Password").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegisterer prometheus.Registerer, storageApp absto.Storage, tracerApp tracer.App) App {
	imaginaryReq := request.Post(*config.imaginaryURL).WithClient(slowClient).BasicAuth(strings.TrimSpace(*config.imaginaryUser), *config.imaginaryPass)

	return App{
		tmpFolder:          *config.tmpFolder,
		storageApp:         storageApp,
		streamRequestQueue: make(chan model.Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
		metric:             prom.CounterVec(prometheusRegisterer, "vith", "", "item", "source", "kind", "type", "state"),
		imaginaryReq:       imaginaryReq,
		tracer:             tracerApp.GetTracer("vith"),
	}
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
