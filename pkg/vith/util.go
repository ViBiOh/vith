package vith

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/vith/pkg/model"
)

var noopFunc = func() {
	// nothing to do
}

var noErrFunc = func() error {
	return nil
}

func (a App) getThumbnailGenerator(itemType model.ItemType) func(context.Context, string, string, uint64) error {
	switch itemType {
	case model.TypeVideo:
		return a.videoThumbnail
	case model.TypeImage:
		return a.imageThumbnail
	default:
		return func(_ context.Context, _, _ string, _ uint64) error {
			return fmt.Errorf("unknown generator for `%s`", itemType)
		}
	}
}

func (a App) getInputName(ctx context.Context, name string) (string, func(), error) {
	switch a.storageApp.Name() {
	case filesystem.Name:
		return a.storageApp.Path(name), noopFunc, nil

	case s3.Name:
		var reader io.ReadCloser
		reader, err := a.storageApp.ReadFrom(ctx, name)
		if err != nil {
			return "", noopFunc, fmt.Errorf("read from storage: %w", err)
		}

		localName, err := a.saveFileLocally(reader, fmt.Sprintf("input_%s", name))
		if err != nil {
			cleanLocalFile(localName)
			return "", noopFunc, fmt.Errorf("save file locally: %w", err)
		}

		return localName, func() { cleanLocalFile(localName) }, nil

	default:
		return "", noopFunc, errors.New("unknown storage provider")
	}
}

func (a App) getOutputName(ctx context.Context, name string) (string, func() error) {
	switch a.storageApp.Name() {
	case filesystem.Name:
		return a.storageApp.Path(name), noErrFunc

	case s3.Name:
		localName := a.getLocalFilename(fmt.Sprintf("output_%s", name))

		return localName, func() error {
			defer cleanLocalFile(localName)
			return a.copyAndCloseLocalFile(ctx, localName, name)
		}

	default:
		return "", func() error { return errors.New("unknown storage provider") }
	}
}

func (a App) getLocalFilename(name string) string {
	return filepath.Join(a.tmpFolder, sha.New(name))
}

func (a App) saveFileLocally(input io.ReadCloser, name string) (string, error) {
	defer closeWithLog(input, "saveFileLocally", "input")

	outputName := a.getLocalFilename(name)

	writer, err := os.OpenFile(outputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer closeWithLog(writer, "saveFileLocally", outputName)

	_, err = io.Copy(writer, input)
	return outputName, err
}

func (a App) copyAndCloseLocalFile(ctx context.Context, src, target string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat local file `%s`: %w", src, err)
	}

	input, err := os.OpenFile(src, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open local file `%s`: %w", src, err)
	}
	defer closeWithLog(input, "copyLocalFile", "input")

	if err := a.storageApp.WriteSizedTo(ctx, target, info.Size(), input); err != nil {
		return fmt.Errorf("write to storage: %w", err)
	}

	return nil
}

func copyLocalFile(name string, output io.Writer) error {
	input, err := os.OpenFile(name, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer closeWithLog(input, "copyLocalFile", "input")

	_, err = io.Copy(output, input)
	if err != nil {
		return fmt.Errorf("copy local file: %w", err)
	}

	return nil
}

func cleanLocalFile(name string) {
	if len(name) == 0 {
		return
	}

	if removeErr := os.Remove(name); removeErr != nil {
		logger.Warn("remove file `%s`: %s", name, removeErr)
	}
}

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", fn).WithField("item", item).Error("close: %s", err)
	}
}
