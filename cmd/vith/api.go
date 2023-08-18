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

	telemetryApp, err := telemetry.New(ctx, telemetryConfig)
	if err != nil {
		slog.Error("create telemetry", "err", err)
		os.Exit(1)
	}

	request.AddOpenTelemetryToDefaultClient(telemetryApp.GetMeterProvider(), telemetryApp.GetTraceProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	healthApp := health.New(healthConfig)

	storageProvider, err := absto.New(abstoConfig, telemetryApp.GetTracer("storage"))
	if err != nil {
		slog.Error("create storage", "err", err)
		os.Exit(1)
	}

	meter := telemetryApp.GetMeter("github.com/ViBiOh/vith/cmd/vith")

	amqpClient, err := amqp.New(amqpConfig, meter, telemetryApp.GetTracer("amqp"))
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		slog.Error("create amqp", "err", err)
		os.Exit(1)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	vithApp := vith.New(vithConfig, amqpClient, storageProvider, meter, telemetryApp.GetTracer("vith"))

	streamHandlerApp, err := amqphandler.New(streamHandlerConfig, amqpClient, telemetryApp.GetTracer("amqp_handler"), vithApp.AmqpStreamHandler)
	if err != nil {
		slog.Error("create amqp handler stream", "err", err)
		os.Exit(1)
	}

	thumbnailHandlerApp, err := amqphandler.New(thumbnailHandlerConfig, amqpClient, telemetryApp.GetTracer("amqp_handler"), vithApp.AmqpThumbnailHandler)
	if err != nil {
		slog.Error("create amqp handler", "err", err)
		os.Exit(1)
	}

	doneCtx := healthApp.Done(ctx)
	endCtx := healthApp.End(ctx)

	go streamHandlerApp.Start(doneCtx)
	go thumbnailHandlerApp.Start(doneCtx)
	go vithApp.Start(doneCtx)

	go appServer.Start(endCtx, "http", httputils.Handler(vithApp.Handler(), healthApp, recoverer.Middleware, telemetryApp.Middleware("http")))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), vithApp.Done(), streamHandlerApp.Done(), thumbnailHandlerApp.Done())
}
