package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/vith/pkg/vith"
)

type service struct {
	streamHandler    *amqphandler.Service
	thumbnailHandler *amqphandler.Service
	vith             vith.Service
	server           server.Server
}

func newService(config configuration, clients clients, adapters adapters) (*service, error) {
	vithService := vith.New(config.vith, clients.amqp, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	streamHandlerService, err := amqphandler.New(config.streamHandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), vithService.AmqpStreamHandler)
	if err != nil {
		return nil, fmt.Errorf("streamHandler: %w", err)
	}

	thumbnailHandlerService, err := amqphandler.New(config.streamHandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), vithService.AmqpThumbnailHandler)
	if err != nil {
		return nil, fmt.Errorf("thumbnailHandlerService: %w", err)
	}

	return &service{
		vith:             vithService,
		streamHandler:    streamHandlerService,
		thumbnailHandler: thumbnailHandlerService,
		server:           server.New(config.server),
	}, nil
}

func (s *service) Start(ctx context.Context) {
	go s.streamHandler.Start(ctx)
	go s.thumbnailHandler.Start(ctx)
	s.vith.Start(ctx)
}
