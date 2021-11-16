package vith

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	if req.ItemType != model.TypeVideo {
		return errors.New("stream are possible for video type only")
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	if info, err := os.Stat(req.Output); err != nil || !info.IsDir() {
		return fmt.Errorf("output `%s` doesn't exist or is not a directory", req.Output)
	}

	if err := a.generateStream(req); err != nil {
		return fmt.Errorf("unable to generate stream: %s", err)
	}

	return nil
}

// AmqpThumbnailHandler for amqp thumbnail request
func (a App) AmqpThumbnailHandler(message amqp.Delivery) error {
	if !a.hasDirectAccess() {
		return errors.New("vith has no direct access to filesystem")
	}

	var req model.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	req.Input = filepath.Join(a.workingDir, req.Input)
	req.Output = filepath.Join(a.workingDir, req.Output)

	if info, err := os.Stat(req.Input); err != nil || info.IsDir() {
		return fmt.Errorf("input `%s` doesn't exist or is a directory", req.Input)
	}

	if req.ItemType == model.TypePDF {
		if err := a.pdf(req); err != nil {
			return err
		}
	}

	if err := thumbnail(req); err != nil {
		return fmt.Errorf("unable to generate thumbnail: %s", err)
	}

	return nil
}
