package vith

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/vith/pkg/model"
)

const thumbnailDuration = 5

func (a App) storageThumbnail(ctx context.Context, itemType model.ItemType, input, output string, scale uint64) (err error) {
	if err = a.storageApp.CreateDir(ctx, path.Dir(output)); err != nil {
		err = fmt.Errorf("create directory for output: %w", err)
		return
	}

	if itemType == model.TypePDF {
		err = a.streamPdf(ctx, input, output, scale)
		return
	}

	var inputName string
	var finalizeInput func()

	inputName, finalizeInput, err = a.getInputName(ctx, input)
	if err != nil {
		err = fmt.Errorf("get input name: %w", err)
	} else {
		outputName, finalizeOutput := a.getOutputName(ctx, output)
		err = httpModel.WrapError(a.getThumbnailGenerator(itemType)(ctx, inputName, outputName, scale), finalizeOutput())
		finalizeInput()
	}

	return err
}

func (a App) streamPdf(ctx context.Context, name, output string, scale uint64) error {
	reader, err := a.storageApp.ReadFrom(ctx, name)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}

	done := make(chan error)
	outputReader, outputWriter := io.Pipe()

	go func() {
		defer close(done)

		var err error

		var item absto.Item
		item, err = a.storageApp.Info(ctx, name)
		if err != nil {
			err = fmt.Errorf("stat input file: %w", err)
		} else {
			err = a.pdfThumbnail(ctx, reader, outputWriter, item.Size, scale)
		}

		if closeErr := outputWriter.Close(); closeErr != nil {
			err = httpModel.WrapError(err, closeErr)
		}

		done <- err
	}()

	err = a.storageApp.WriteTo(ctx, output, outputReader, absto.WriteOpts{})
	if thumbnailErr := <-done; thumbnailErr != nil {
		err = httpModel.WrapError(err, thumbnailErr)
	}

	if closeErr := outputReader.Close(); closeErr != nil {
		err = httpModel.WrapError(err, closeErr)
	}

	if err != nil {
		if removeErr := a.storageApp.Remove(ctx, output); removeErr != nil {
			err = httpModel.WrapError(err, fmt.Errorf("remove: %w", removeErr))
		}
	}

	return err
}

func (a App) pdfThumbnail(ctx context.Context, input io.ReadCloser, output io.Writer, contentLength int64, scale uint64) error {
	r, err := a.imaginaryReq.Path("/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=webp", scale, scale).Build(ctx, input)
	if err != nil {
		defer closeWithLog(input, "pdfThumbnail", "")
		return fmt.Errorf("build request: %w", err)
	}

	r.ContentLength = contentLength

	resp, err := request.DoWithClient(slowClient, r)
	if err != nil {
		return fmt.Errorf("request imaginary: %w", err)
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err = io.CopyBuffer(output, resp.Body, buffer.Bytes()); err != nil {
		return fmt.Errorf("copy imaginary response: %w", err)
	}

	return nil
}

func (a App) imageThumbnail(ctx context.Context, inputName, outputName string, scale uint64) error {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "ffmpeg_thumbnail")
	defer end()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputName, "-map_metadata", "-1", "-vf", fmt.Sprintf("crop='min(iw,ih)':'min(iw,ih)',scale=%d:%d", scale, scale), "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "75", "-an", "-preset", "picture", "-y", "-f", "webp", "-frames:v", "1", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		cleanLocalFile(outputName)
		return fmt.Errorf("ffmpeg image: %s: %w", buffer.String(), err)
	}

	return nil
}

func (a App) videoThumbnail(ctx context.Context, inputName, outputName string, scale uint64) error {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "ffmpeg_video_thumbnail")
	defer end()

	var ffmpegOpts []string
	var customOpts []string

	if _, duration, err := a.getVideoDetailsFromLocal(ctx, inputName); err != nil {
		logger.Error("get container duration for `%s`: %s", inputName, err)
		ffmpegOpts = append(ffmpegOpts, "-ss", "1.000")
	} else {
		startPoint := duration / 2
		if duration > thumbnailDuration {
			startPoint -= thumbnailDuration / 2
		}

		ffmpegOpts = append(ffmpegOpts, "-ss", fmt.Sprintf("%.3f", startPoint))
	}

	format := fmt.Sprintf("crop='min(iw,ih)':'min(iw,ih)',scale=%d:%d", scale, scale)
	if scale == SmallSize {
		ffmpegOpts = append(ffmpegOpts, "-t", strconv.Itoa(thumbnailDuration))
		customOpts = []string{"-r", "8", "-loop", "0"}
	} else {
		customOpts = []string{"-frames:v", "1"}
	}

	ffmpegOpts = append(ffmpegOpts, "-i", inputName, "-map_metadata", "-1", "-vf", format, "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "75", "-an", "-preset", "picture", "-y", "-f", "webp")
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
		return fmt.Errorf("ffmpeg video: %s: %w", buffer.String(), err)
	}

	return nil
}

func (a App) getVideoDetailsFromLocal(ctx context.Context, name string) (int64, float64, error) {
	reader, err := os.OpenFile(name, os.O_RDONLY, 0o600)
	if err != nil {
		return 0, 0, fmt.Errorf("open file: %w", err)
	}
	defer closeWithLog(reader, "getVideoBitrate", name)

	return a.getVideoDetails(ctx, name)
}
