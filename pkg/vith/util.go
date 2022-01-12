package vith

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

func (a App) getInputVideoName(name string) (string, func(), error) {
	switch a.storageApp.Name() {
	case filesystem.Name:
		return a.storageApp.Path(name), func() {}, nil

	case s3.Name:
		var reader io.ReadCloser
		reader, err := a.storageApp.ReaderFrom(name)
		if err != nil {
			return "", func() {}, fmt.Errorf("unable to read from storage: %s", err)
		}

		localName, err := a.saveFileLocally(reader, fmt.Sprintf("input_%s", name))
		if err != nil {
			cleanLocalFile(localName)
			return "", func() {}, fmt.Errorf("unable to save file locally: %s", err)
		}

		return localName, func() { cleanLocalFile(localName) }, nil

	default:
		return "", func() {}, errors.New("unknown storage provider")
	}
}

func (a App) getOutputVideoName(name string) (string, func() error) {
	switch a.storageApp.Name() {
	case filesystem.Name:
		return a.storageApp.Path(name), func() error { return nil }

	case s3.Name:
		outputName := a.getLocalFilename(fmt.Sprintf("output_%s", name))

		return outputName, func() error {
			writer, err := a.storageApp.WriterTo(name)
			if err != nil {
				return fmt.Errorf("unable to open writer to storage: %s", err)
			}
			defer closeWithLog(writer, "getOutputVideoName", name)

			if err = copyLocalFile(outputName, writer); err != nil {
				return fmt.Errorf("unable to write to storage: %s", err)
			}

			cleanLocalFile(outputName)

			return nil
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
		return "", fmt.Errorf("unable to open file: %s", err)
	}
	defer closeWithLog(writer, "saveFileLocally", outputName)

	_, err = io.Copy(writer, input)
	return outputName, err
}

func copyLocalFile(name string, output io.Writer) error {
	input, err := os.OpenFile(name, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open local file: %s", err)
	}
	defer closeWithLog(input, "copyLocalFile", "input")

	_, err = io.Copy(output, input)
	if err != nil {
		return fmt.Errorf("unable to copy local file: %s", err)
	}

	return nil
}

func cleanLocalFile(name string) {
	if len(name) == 0 {
		return
	}

	if removeErr := os.Remove(name); removeErr != nil {
		logger.Warn("unable to remove file `%s`: %s", name, removeErr)
	}
}

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", fn).WithField("item", item).Error("unable to close: %s", err)
	}
}
