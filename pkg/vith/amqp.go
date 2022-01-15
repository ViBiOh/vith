package vith

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
)

// AmqpStreamHandler for amqp stream request
func (a App) AmqpStreamHandler(message amqp.Delivery) error {
	if !a.storageApp.Enabled() {
		return errors.New("vith has no direct access to filesystem")
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

	if err := a.generateStream(req); err != nil {
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

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if err := a.storageApp.CreateDir(path.Dir(req.Output)); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("unable to create directory for output: %s", err)
	}

	if req.ItemType == model.TypeVideo {
		inputName, finalizeInput, err := a.getInputVideoName(req.Input)
		if err != nil {
			a.increaseMetric("http", "amqp", "video", "error")
			return fmt.Errorf("unable to get input video name: %s", err)
		}
		defer finalizeInput()

		outputName, finalizeOutput := a.getOutputVideoName(req.Output)

		if err := a.videoThumbnail(inputName, outputName); err != nil {
			return fmt.Errorf("unable to generate video thumbnail: %s", err)
		}

		return finalizeOutput()
	}

	writer, closer, err := a.storageApp.WriterTo(req.Output)
	if err != nil {
		err = fmt.Errorf("unable to open writer to storage: %s", err)
	} else {
		defer closerWithLog(closer, "AmqpThumbnailHandler", req.Output)
		err = a.streamThumbnail(req.Input, writer, req.ItemType)
	}

	if err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
	} else {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "success")
	}

	return err
}
