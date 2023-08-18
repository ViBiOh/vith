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

func (a App) AmqpStreamHandler(ctx context.Context, message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, a.tracer, "amqp")
	defer end(&err)

	var req model.Request
	if err = json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric(ctx, "amqp", "stream", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if req.ItemType != model.TypeVideo {
		a.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "forbidden")
		return errors.New("stream are possible for video type only")
	}

	if len(req.Input) == 0 {
		a.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "input_invalid")
		return errors.New("input is mandatory")
	}

	if len(req.Output) == 0 {
		a.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "output_invalid")
		return errors.New("output is mandatory")
	}

	if err = a.storageApp.Mkdir(ctx, path.Dir(req.Output), absto.DirectoryPerm); err != nil {
		a.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("create directory for output: %w", err)
	}

	if err = a.generateStream(ctx, req); err != nil {
		a.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "error")
		return fmt.Errorf("generate stream: %w", err)
	}

	a.increaseMetric(ctx, "amqp", "stream", req.ItemType.String(), "success")

	return nil
}

func (a App) AmqpThumbnailHandler(ctx context.Context, message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, a.tracer, "amqp")
	defer end(&err)

	var req model.Request
	if err = json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric(ctx, "amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("parse payload: %w", err)
	}

	if err = a.storageThumbnail(ctx, req.ItemType, req.Input, req.Output, req.Scale); err != nil {
		a.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "error")
		return err
	}

	if err = a.amqpClient.PublishJSON(ctx, req, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("publish amqp message: %w", err)
	}

	a.increaseMetric(ctx, "amqp", "thumbnail", req.ItemType.String(), "success")
	return nil
}
