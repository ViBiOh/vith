package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/vith/pkg/vith"
)

func main() {
	fs := flag.NewFlagSet("vith", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", "2m"), flags.NewOverride("WriteTimeout", "2m"))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
	pprofServerConfig := server.Flags(fs, "pprof", flags.NewOverride("Port", 9999))
	healthConfig := health.Flags(fs, "")

	tmpFolder := flags.New("", "vith").Name("TmpFolder").Default("/tmp").Label("Folder used for temporary files storage").ToString(fs)

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	pprofServer := server.New(pprofServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	go pprofServer.Start("pprof", healthApp.End(), http.DefaultServeMux)
	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(vith.Handler(*tmpFolder), healthApp, recoverer.Middleware, prometheusApp.Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done())
}
