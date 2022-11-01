package vith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
)

func (a App) AmqpStreamHandler(ctx context.Context, message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end()

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "stream", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if req.ItemType != model.TypeVideo {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "forbidden")
		return errors.New("stream are possible for video type only")
	}

	if len(req.Input) == 0 {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "input_invalid")
		return errors.New("input is mandatory")
	}

	if len(req.Output) == 0 {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "output_invalid")
		return errors.New("output is mandatory")
	}

	if err := a.storageApp.CreateDir(ctx, path.Dir(req.Output)); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("create directory for output: %w", err)
	}

	if err := a.generateStream(ctx, req); err != nil {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "error")
		return fmt.Errorf("generate stream: %w", err)
	}

	a.increaseMetric("amqp", "stream", req.ItemType.String(), "success")

	return nil
}

func (a App) AmqpThumbnailHandler(ctx context.Context, message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end()

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if err := a.storageThumbnail(ctx, req.ItemType, req.Input, req.Output, req.Scale); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return err
	}

	if err := a.amqpClient.PublishJSON(ctx, req, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("publish amqp message: %w", err)
	}

	a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "success")
	return nil
}
