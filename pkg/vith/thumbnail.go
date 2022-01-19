package vith

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) storageThumbnail(itemType model.ItemType, input, output string) (err error) {
	if err := a.storageApp.CreateDir(path.Dir(output)); err != nil {
		return fmt.Errorf("unable to create directory for output: %s", err)
	}

	switch itemType {
	case model.TypeVideo:
		var inputName string
		var finalizeInput func()

		inputName, finalizeInput, err = a.getInputVideoName(input)
		if err != nil {
			err = fmt.Errorf("unable to get input video name: %s", err)
		} else {
			defer finalizeInput()

			outputName, finalizeOutput := a.getOutputVideoName(output)
			err = httpModel.WrapError(a.videoThumbnail(inputName, outputName), finalizeOutput())
		}

	default:
		err = a.streamThumbnail(input, output, itemType)
	}

	return err
}

func (a App) streamThumbnail(name, output string, itemType model.ItemType) error {
	reader, err := a.storageApp.ReadFrom(name)
	if err != nil {
		return fmt.Errorf("unable to open input file: %s", err)
	}

	// PDF file are closed by request sender
	if itemType != model.TypePDF {
		defer closeWithLog(reader, "streamThumbnail", name)
	}

	done := make(chan error)
	outputReader, outputWriter := io.Pipe()

	go func() {
		defer close(done)

		var err error

		switch itemType {
		case model.TypePDF:
			var item absto.Item
			item, err = a.storageApp.Info(name)
			if err != nil {
				err = fmt.Errorf("unable to stat input file: %s", err)
			} else {
				err = a.pdfThumbnail(reader, outputWriter, item.Size)
			}

		case model.TypeImage:
			err = imageThumbnail(reader, outputWriter)

		default:
			err = fmt.Errorf("unhandled itemType `%s` for streaming thumbnail", itemType)
		}

		if closeErr := outputWriter.Close(); closeErr != nil {
			err = httpModel.WrapError(err, closeErr)
		}

		done <- err
	}()

	err = a.storageApp.WriteTo(output, outputReader)
	if thumbnailErr := <-done; thumbnailErr != nil {
		err = httpModel.WrapError(err, thumbnailErr)
	}

	if closeErr := outputReader.Close(); closeErr != nil {
		err = httpModel.WrapError(err, closeErr)
	}

	if err != nil {
		if removeErr := a.storageApp.Remove(output); removeErr != nil {
			err = httpModel.WrapError(err, fmt.Errorf("unable to remove: %s", removeErr))
		}
	}

	return err
}

func (a App) pdfThumbnail(input io.ReadCloser, output io.Writer, contentLength int64) error {
	r, err := a.imaginaryReq.Build(context.Background(), input)
	if err != nil {
		defer closeWithLog(input, "pdfThumbnail", "")
		return fmt.Errorf("unable to build request: %s", err)
	}

	r.ContentLength = contentLength

	resp, err := request.DoWithClient(slowClient, r)
	if err != nil {
		return fmt.Errorf("unable to request imaginary: %s", err)
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err = io.CopyBuffer(output, resp.Body, buffer.Bytes()); err != nil {
		return fmt.Errorf("unable to copy imaginary response: %s", err)
	}

	return nil
}

func imageThumbnail(input io.Reader, output io.Writer) error {
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150", "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture", "-f", "webp", "pipe:1")

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", buffer.String(), err)
	}

	return nil
}

func (a App) streamVideoThumbnail(inputName string, output io.Writer) error {
	outputName := a.getLocalFilename(fmt.Sprintf("output_%s", inputName))

	if err := a.videoThumbnail(inputName, outputName); err != nil {
		return fmt.Errorf("unable to generate video thumbnail: %s", err)
	}

	defer cleanLocalFile(outputName)

	if err := copyLocalFile(outputName, output); err != nil {
		return fmt.Errorf("unable to copy video thumbnail: %s", err)
	}

	return nil
}

func (a App) videoThumbnail(inputName, outputName string) error {
	var ffmpegOpts []string
	var customOpts []string

	if _, duration, err := getVideoDetailsFromLocal(inputName); err != nil {
		logger.Error("unable to get container duration for `%s`: %s", inputName, err)

		ffmpegOpts = append(ffmpegOpts, "-ss", "1.000")
		customOpts = []string{"-frames:v", "1"}
	} else {
		ffmpegOpts = append(ffmpegOpts, "-ss", fmt.Sprintf("%.3f", duration/2), "-t", "5")
		customOpts = []string{"-vsync", "0", "-loop", "0"}
	}

	ffmpegOpts = append(ffmpegOpts, "-i", inputName, "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150,fps=10", "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture", "-f", "webp")
	ffmpegOpts = append(ffmpegOpts, customOpts...)
	ffmpegOpts = append(ffmpegOpts, outputName)
	cmd := exec.Command("ffmpeg", ffmpegOpts...)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		cleanLocalFile(outputName)
		return fmt.Errorf("%s: %s", buffer.String(), err)
	}

	return nil
}

func getVideoDetailsFromLocal(name string) (int64, float64, error) {
	reader, err := os.OpenFile(name, os.O_RDONLY, 0o600)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open file: %s", err)
	}
	defer closeWithLog(reader, "getVideoBitrate", name)

	return getVideoDetails(reader)
}
