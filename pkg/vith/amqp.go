package vith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/trace"
)

// AmqpStreamHandler for amqp stream request
func (a App) AmqpStreamHandler(message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	ctx := context.Background()

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "amqp")
		defer span.End()
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "stream", "", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
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

	if err := a.storageApp.CreateDir(path.Dir(req.Output)); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("unable to create directory for output: %s", err)
	}

	if err := a.generateStream(ctx, req); err != nil {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "error")
		return fmt.Errorf("unable to generate stream: %s", err)
	}

	a.increaseMetric("amqp", "stream", req.ItemType.String(), "success")

	return nil
}

// AmqpThumbnailHandler for amqp thumbnail request
func (a App) AmqpThumbnailHandler(message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
	}

	ctx := context.Background()

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "amqp")
		defer span.End()
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if err := a.storageThumbnail(ctx, req.ItemType, req.Input, req.Output, req.Scale); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return err
	}

	a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "success")
	return nil
}
