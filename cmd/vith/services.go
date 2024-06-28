package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/vith/pkg/vith"
)

type services struct {
	server           *server.Server
	streamHandler    *amqphandler.Service
	thumbnailHandler *amqphandler.Service
	vith             vith.Service
}

func newServices(config configuration, clients clients, adapters adapters) (services, error) {
	var output services
	var err error

	output.server = server.New(config.server)

	output.vith = vith.New(config.vith, clients.amqp, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	output.streamHandler, err = amqphandler.New(config.streamHandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.vith.AmqpStreamHandler)
	if err != nil {
		return output, fmt.Errorf("stream: %w", err)
	}

	output.thumbnailHandler, err = amqphandler.New(config.thumbnailHandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.vith.AmqpThumbnailHandler)
	if err != nil {
		return output, fmt.Errorf("thumbnail: %w", err)
	}

	return output, nil
}

func (s services) Start(ctx context.Context) {
	go s.streamHandler.Start(ctx)
	go s.thumbnailHandler.Start(ctx)
	go s.vith.Start(ctx)
}
