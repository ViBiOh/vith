package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/vith"
)

type configuration struct {
	alcotest  *alcotest.Config
	logger    *logger.Config
	telemetry *telemetry.Config
	pprof     *pprof.Config
	server    *server.Config
	health    *health.Config

	vith             *vith.Config
	absto            *absto.Config
	amqp             *amqp.Config
	streamHandler    *amqphandler.Config
	thumbnailHandler *amqphandler.Config
}

func newConfig() configuration {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	config := configuration{
		server:    server.Flags(fs, ""),
		health:    health.Flags(fs, ""),
		alcotest:  alcotest.Flags(fs, ""),
		logger:    logger.Flags(fs, "logger"),
		telemetry: telemetry.Flags(fs, "telemetry"),
		pprof:     pprof.Flags(fs, "pprof"),

		vith:             vith.Flags(fs, ""),
		absto:            absto.Flags(fs, "storage", flags.NewOverride("FileSystemDirectory", "")),
		amqp:             amqp.Flags(fs, "amqp"),
		streamHandler:    amqphandler.Flags(fs, "stream", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "stream"), flags.NewOverride("RoutingKey", "stream")),
		thumbnailHandler: amqphandler.Flags(fs, "thumbnail", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "thumbnail"), flags.NewOverride("RoutingKey", "thumbnail")),
	}

	_ = fs.Parse(os.Args[1:])

	return config
}
