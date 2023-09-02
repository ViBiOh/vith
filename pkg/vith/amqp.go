package vith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/model"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (s Service) AmqpStreamHandler(ctx context.Context, message amqp.Delivery) error {
	if !s.storage.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "amqp")
	defer end(&err)

	var req model.Request
	if err = json.Unmarshal(message.Body, &req); err != nil {
		s.increaseMetric(ctx, "amqp", "stream", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if req.ItemType != model.TypeVideo {
		s.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "forbidden")
		return errors.New("stream are possible for video type only")
	}

	if len(req.Input) == 0 {
		s.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "input_invalid")
		return errors.New("input is mandatory")
	}

	if len(req.Output) == 0 {
		s.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "output_invalid")
		return errors.New("output is mandatory")
	}

	if err = s.storage.Mkdir(ctx, path.Dir(req.Output), absto.DirectoryPerm); err != nil {
		s.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("create directory for output: %w", err)
	}

	if err = s.generateStream(ctx, req); err != nil {
		s.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "error")
		return fmt.Errorf("generate stream: %w", err)
	}

	s.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "success")

	return nil
}

func (s Service) AmqpThumbnailHandler(ctx context.Context, message amqp.Delivery) error {
	if !s.storage.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "amqp")
	defer end(&err)

	var req model.Request
	if err = json.Unmarshal(message.Body, &req); err != nil {
		s.increaseMetric(ctx, "amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if err = s.storageThumbnail(ctx, req.ItemType, req.Input, req.Output, req.Scale); err != nil {
		s.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "error")
		return err
	}

	if err = s.amqpClient.PublishJSON(ctx, req, s.amqpExchange, s.amqpRoutingKey); err != nil {
		return fmt.Errorf("publish amqp message: %w", err)
	}

	s.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "success")
	return nil
}
