package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/vith"
)

func main() {
	fs := flag.NewFlagSet("vith", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", 2*time.Minute), flags.NewOverride("WriteTimeout", 2*time.Minute))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	telemetryConfig := telemetry.Flags(fs, "telemetry")

	vithConfig := vith.Flags(fs, "")
	abstoConfig := absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", ""))

	amqpConfig := amqp.Flags(fs, "amqp")
	streamHandlerConfig := amqphandler.Flags(fs, "stream", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "stream"), flags.NewOverride("RoutingKey", "stream"))
	thumbnailHandlerConfig := amqphandler.Flags(fs, "thumbnail", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "thumbnail"), flags.NewOverride("RoutingKey", "thumbnail"))

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	alcotest.DoAndExit(alcotestConfig)

	logger.Init(loggerConfig)

	ctx := context.Background()

	telemetryService, err := telemetry.New(ctx, telemetryConfig)
	if err != nil {
		slog.ErrorContext(ctx, "create telemetry", "error", err)
		os.Exit(1)
	}

	defer telemetryService.Close(ctx)

	logger.AddOpenTelemetryToDefaultLogger(telemetryService)
	request.AddOpenTelemetryToDefaultClient(telemetryService.MeterProvider(), telemetryService.TracerProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	healthService := health.New(ctx, healthConfig)

	storageProvider, err := absto.New(abstoConfig, telemetryService.TracerProvider())
	if err != nil {
		slog.ErrorContext(ctx, "create storage", "error", err)
		os.Exit(1)
	}

	amqpClient, err := amqp.New(amqpConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		slog.ErrorContext(ctx, "create amqp", "error", err)
		os.Exit(1)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	vithService := vith.New(vithConfig, amqpClient, storageProvider, telemetryService.MeterProvider(), telemetryService.TracerProvider())

	streamHandlerService, err := amqphandler.New(streamHandlerConfig, amqpClient, telemetryService.MeterProvider(), telemetryService.TracerProvider(), vithService.AmqpStreamHandler)
	if err != nil {
		slog.ErrorContext(ctx, "create amqp handler stream", "error", err)
		os.Exit(1)
	}

	thumbnailHandlerService, err := amqphandler.New(thumbnailHandlerConfig, amqpClient, telemetryService.MeterProvider(), telemetryService.TracerProvider(), vithService.AmqpThumbnailHandler)
	if err != nil {
		slog.ErrorContext(ctx, "create amqp handler", "error", err)
		os.Exit(1)
	}

	doneCtx := healthService.DoneCtx()
	endCtx := healthService.EndCtx()

	go streamHandlerService.Start(doneCtx)
	go thumbnailHandlerService.Start(doneCtx)
	go vithService.Start(doneCtx)

	go appServer.Start(endCtx, httputils.Handler(vithService.Handler(), healthService, recoverer.Middleware, telemetryService.Middleware("http")))

	healthService.WaitForTermination(appServer.Done())

	server.GracefulWait(appServer.Done(), vithService.Done(), streamHandlerService.Done(), thumbnailHandlerService.Done())
}
