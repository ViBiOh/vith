package vith

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
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
		a.increaseMetric("amqp", "stream", "", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if req.ItemType != model.TypeVideo {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "forbidden")
		return errors.New("stream are possible for video type only")
	}

	if len(req.Input) == 0 || strings.Contains(req.Input, "..") {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "input_invalid")
		return errors.New("input is mandatory or contains `..`")
	}

	if len(req.Output) == 0 || strings.Contains(req.Output, "..") {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "output_invalid")
		return errors.New("output is mandatory or contains `..`")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		a.increaseMetric("amqp", "stream", req.ItemType.String(), "not_found")
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	if _, err := os.Stat(req.Output); err == nil {
		logger.Info("Stream for %s already exists, skipping.", req.Input)
		return nil
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
	if !a.hasDirectAccess() {
		return errors.New("vith has no direct access to filesystem")
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		a.increaseMetric("amqp", "thumbnail", "", "invalid")
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if len(req.Input) == 0 || strings.Contains(req.Input, "..") {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "input_invalid")
		return errors.New("input is mandatory or contains `..`")
	}

	if len(req.Output) == 0 || strings.Contains(req.Output, "..") {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "output_invalid")
		return errors.New("output is mandatory or contains `..`")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "not_found")
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	if _, err := os.Stat(req.Output); err == nil {
		logger.Info("Thumbnail for %s already exists, skipping.", req.Input)
		return nil
	}

	dirname := path.Dir(req.Output)
	if _, err := os.Stat(dirname); err != nil {
		if !os.IsNotExist(err) {
			a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "dir_error")
			return fmt.Errorf("unable to stat output directory: %s", err)
		}

		if err = os.MkdirAll(dirname, 0o700); err != nil {
			a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "dir_error")
			return fmt.Errorf("unable to create output directory: %s", err)
		}
	}

	if req.ItemType == model.TypePDF {
		if err := a.pdf(req); err != nil {
			a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
			return fmt.Errorf("unable to generate pdf: %s", err)
		}

		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "success")
		return nil
	}

	if err := thumbnail(req); err != nil {
		a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "error")
		return fmt.Errorf("unable to generate thumbnail: %s", err)
	}

	a.increaseMetric("amqp", "thumbnail", req.ItemType.String(), "success")

	return nil
}
