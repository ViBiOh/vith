package vith

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
)

// AmqpStreamHandler for amqp stream request
func (a App) AmqpStreamHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("vith has no direct access to filesystem")
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "stream", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if req.ItemType != model.TypeVideo {
		return errors.New("stream are possible for video type only")
	}

	if len(req.Input) == 0 || strings.Contains(req.Input, "..") {
		a.increaseMetric("amqp", "stream", "input_invalid")
		return errors.New("input is mandatory or contains `..`")
	}

	if len(req.Output) == 0 || strings.Contains(req.Output, "..") {
		a.increaseMetric("amqp", "stream", "output_invalid")
		return errors.New("output is mandatory or contains `..`")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		a.increaseMetric("amqp", "stream", "not_found")
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	if err := a.generateStream(req); err != nil {
		a.increaseMetric("amqp", "stream", "error")
		return fmt.Errorf("unable to generate stream: %s", err)
	}

	a.increaseMetric("amqp", "stream", "success")

	return nil
}

// AmqpThumbnailHandler for amqp thumbnail request
func (a App) AmqpThumbnailHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("vith has no direct access to filesystem")
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if len(req.Input) == 0 || strings.Contains(req.Input, "..") {
		a.increaseMetric("amqp", "thumbnail", "input_invalid")
		return errors.New("input is mandatory or contains `..`")
	}

	if len(req.Output) == 0 || strings.Contains(req.Output, "..") {
		a.increaseMetric("amqp", "thumbnail", "output_invalid")
		return errors.New("output is mandatory or contains `..`")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		a.increaseMetric("amqp", "thumbnail", "not_found")
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	a.increaseMetric("amqp", "thumbnail", req.ItemType.String())

	if req.ItemType == model.TypePDF {
		if err := a.pdf(req); err != nil {
			a.increaseMetric("amqp", "thumbnail", "error")
			return fmt.Errorf("unable to generate pdf: %s", err)
		}
		return nil
	}

	if err := thumbnail(req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "error")
		return fmt.Errorf("unable to generate thumbnail: %s", err)
	}

	return nil
}
