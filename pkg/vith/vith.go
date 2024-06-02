package vith

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
	"net/http"
	"sync"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/vith/pkg/model"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	SmallSize = 150

	hlsExtension = ".m3u8"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

type Config struct {
	TmpFolder string

	AmqpExchange   string
	AmqpRoutingKey string
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("TmpFolder", "Folder used for temporary files storage").Prefix(prefix).DocPrefix("vith").StringVar(fs, &config.TmpFolder, "/tmp", overrides)
	flags.New("Exchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.AmqpExchange, "fibr", overrides)
	flags.New("RoutingKey", "AMQP Routing Key to fibr").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.AmqpRoutingKey, "thumbnail_output", overrides)

	return &config
}

type Service struct {
	done               chan struct{}
	stop               chan struct{}
	streamRequestQueue chan model.Request
	storage            absto.Storage
	tracer             trace.Tracer
	amqpClient         *amqp.Client
	metric             metric.Int64Counter
	tmpFolder          string
	amqpExchange       string
	amqpRoutingKey     string
}

func New(config *Config, amqpClient *amqp.Client, storageService absto.Storage, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	service := Service{
		tmpFolder: config.TmpFolder,
		storage:   storageService,

		amqpClient:     amqpClient,
		amqpExchange:   config.AmqpExchange,
		amqpRoutingKey: config.AmqpRoutingKey,

		streamRequestQueue: make(chan model.Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/vith/pkg/vith")

		var err error

		service.metric, err = meter.Int64Counter("vith.item")
		if err != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "create vith counter", slog.Any("error", err))
		}
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("vith")
	}

	return service
}

func (s Service) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			s.handleHead(w, r)
		case http.MethodGet:
			s.handleGet(w, r)
		case http.MethodPost:
			s.handlePost(w, r)
		case http.MethodPut:
			s.handlePut(w, r)
		case http.MethodPatch:
			s.handlePatch(w, r)
		case http.MethodDelete:
			s.handleDelete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
