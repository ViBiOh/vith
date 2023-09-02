package vith

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ViBiOh/absto/pkg/filesystem"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/hash"
	"github.com/ViBiOh/vith/pkg/model"
)

var noopFunc = func() {
	// nothing to do
}

var noErrFunc = func() error {
	return nil
}

func (s Service) getThumbnailGenerator(itemType model.ItemType) func(context.Context, string, string, uint64) error {
	switch itemType {
	case model.TypeVideo:
		return s.videoThumbnail
	case model.TypeImage:
		return s.imageThumbnail
	default:
		return func(_ context.Context, _, _ string, _ uint64) error {
			return fmt.Errorf("unknown generator for `%s`", itemType)
		}
	}
}

func (s Service) getInputName(ctx context.Context, name string) (string, func(), error) {
	switch s.storage.Name() {
	case filesystem.Name:
		return s.storage.Path(name), noopFunc, nil

	case s3.Name:
		var reader io.ReadCloser
		reader, err := s.storage.ReadFrom(ctx, name)
		if err != nil {
			return "", noopFunc, fmt.Errorf("read from storage: %w", err)
		}

		localName, err := s.saveFileLocally(reader, fmt.Sprintf("input_%s", name))
		if err != nil {
			cleanLocalFile(localName)
			return "", noopFunc, fmt.Errorf("save file locally: %w", err)
		}

		return localName, func() { cleanLocalFile(localName) }, nil

	default:
		return "", noopFunc, errors.New("unknown storage provider")
	}
}

func (s Service) getOutputName(ctx context.Context, name string) (string, func() error) {
	switch s.storage.Name() {
	case filesystem.Name:
		return s.storage.Path(name), noErrFunc

	case s3.Name:
		localName := s.getLocalFilename(fmt.Sprintf("output_%s", name))

		return localName, func() error {
			defer cleanLocalFile(localName)
			return s.copyAndCloseLocalFile(ctx, localName, name)
		}

	default:
		return "", func() error { return errors.New("unknown storage provider") }
	}
}

func (s Service) getLocalFilename(name string) string {
	return filepath.Join(s.tmpFolder, hash.String(name))
}

func (s Service) saveFileLocally(input io.ReadCloser, name string) (string, error) {
	defer closeWithLog(input, "saveFileLocally", "input")

	outputName := s.getLocalFilename(name)

	writer, err := os.OpenFile(outputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, absto.RegularFilePerm)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer closeWithLog(writer, "saveFileLocally", outputName)

	_, err = io.Copy(writer, input)
	return outputName, err
}

func (s Service) copyAndCloseLocalFile(ctx context.Context, src, target string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat local file `%s`: %w", src, err)
	}

	input, err := os.OpenFile(src, os.O_RDONLY, absto.RegularFilePerm)
	if err != nil {
		return fmt.Errorf("open local file `%s`: %w", src, err)
	}
	defer closeWithLog(input, "copyLocalFile", "input")

	if err := s.storage.WriteTo(ctx, target, input, absto.WriteOpts{Size: info.Size()}); err != nil {
		return fmt.Errorf("write to storage: %w", err)
	}

	return nil
}

func copyLocalFile(name string, output io.Writer) error {
	input, err := os.OpenFile(name, os.O_RDONLY, absto.RegularFilePerm)
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
		slog.Warn("remove file", "err", removeErr, "name", name)
	}
}

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		slog.Error("close", "err", err, "fn", fn, "item", item)
	}
}
