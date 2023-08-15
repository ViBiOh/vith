package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
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
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/vith/pkg/vith"
)

func main() {
	fs := flag.NewFlagSet("vith", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", 2*time.Minute), flags.NewOverride("WriteTimeout", 2*time.Minute))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", uint(9090)), flags.NewOverride("IdleTimeout", 10*time.Second), flags.NewOverride("ShutdownTimeout", 5*time.Second))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	tracerConfig := tracer.Flags(fs, "tracer")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))

	vithConfig := vith.Flags(fs, "")
	abstoConfig := absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", ""))

	amqpConfig := amqp.Flags(fs, "amqp")
	streamHandlerConfig := amqphandler.Flags(fs, "stream", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "stream"), flags.NewOverride("RoutingKey", "stream"))
	thumbnailHandlerConfig := amqphandler.Flags(fs, "thumbnail", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "thumbnail"), flags.NewOverride("RoutingKey", "thumbnail"))

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	ctx := context.Background()

	tracerApp, err := tracer.New(ctx, tracerConfig)
	logger.Fatal(err)
	defer tracerApp.Close(ctx)
	request.AddTracerToDefaultClient(tracerApp.GetProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	storageProvider, err := absto.New(abstoConfig, tracerApp.GetTracer("storage"))
	logger.Fatal(err)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer(), tracerApp.GetTracer("amqp"))
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Fatal(err)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	vithApp := vith.New(vithConfig, prometheusApp.Registerer(), amqpClient, storageProvider, tracerApp.GetTracer("vith"))

	streamHandlerApp, err := amqphandler.New(streamHandlerConfig, amqpClient, tracerApp.GetTracer("amqp_handler"), vithApp.AmqpStreamHandler)
	logger.Fatal(err)

	thumbnailHandlerApp, err := amqphandler.New(thumbnailHandlerConfig, amqpClient, tracerApp.GetTracer("amqp_handler"), vithApp.AmqpThumbnailHandler)
	logger.Fatal(err)

	doneCtx := healthApp.Done(ctx)
	endCtx := healthApp.End(ctx)

	go streamHandlerApp.Start(doneCtx)
	go thumbnailHandlerApp.Start(doneCtx)
	go vithApp.Start(doneCtx)

	go promServer.Start(endCtx, "prometheus", prometheusApp.Handler())
	go appServer.Start(endCtx, "http", httputils.Handler(vithApp.Handler(), healthApp, recoverer.Middleware, prometheusApp.Middleware, tracerApp.Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), vithApp.Done(), streamHandlerApp.Done(), thumbnailHandlerApp.Done())
}
