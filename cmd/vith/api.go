package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

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

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", "2m"), flags.NewOverride("WriteTimeout", "2m"))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
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

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	tracerApp, err := tracer.New(tracerConfig)
	logger.Fatal(err)
	defer tracerApp.Close()
	request.AddTracerToDefaultClient(tracerApp.GetProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	storageProvider, err := absto.New(abstoConfig)
	logger.Fatal(err)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Error("unable to create amqp client: %s", err)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	vithApp := vith.New(vithConfig, prometheusApp.Registerer(), storageProvider, tracerApp)

	streamHandlerApp, err := amqphandler.New(streamHandlerConfig, amqpClient, vithApp.AmqpStreamHandler)
	logger.Fatal(err)

	thumbnailHandlerApp, err := amqphandler.New(thumbnailHandlerConfig, amqpClient, vithApp.AmqpThumbnailHandler)
	logger.Fatal(err)

	go streamHandlerApp.Start(healthApp.Done())
	go thumbnailHandlerApp.Start(healthApp.Done())
	go vithApp.Start(healthApp.Done())

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(vithApp.Handler(), healthApp, recoverer.Middleware, prometheusApp.Middleware, tracerApp.Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), vithApp.Done(), streamHandlerApp.Done(), thumbnailHandlerApp.Done())
}
