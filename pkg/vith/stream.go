package vith

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/vith/pkg/model"
)

// Done close when work is over
func (a App) Done() <-chan struct{} {
	return a.done
}

// Start worker
func (a App) Start(done <-chan struct{}) {
	defer close(a.done)
	defer close(a.streamRequestQueue)
	defer a.stopOnce()

	if !a.storageApp.Enabled() {
		return
	}

	go func() {
		defer a.stopOnce()

		select {
		case <-done:
		case <-a.done:
		}
	}()

	for req := range a.streamRequestQueue {
		if err := a.generateStream(context.Background(), req); err != nil {
			logger.Error("unable to generate stream: %s", err)
		}
	}
}

func (a App) stopOnce() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
}

func (a App) generateStream(ctx context.Context, req model.Request) error {
	if a.tracer != nil {
		_, span := a.tracer.Start(ctx, "stream")
		defer span.End()
	}

	log := logger.WithField("input", req.Input).WithField("output", req.Output)
	log.Info("Generating stream...")

	inputName, finalizeInput, err := a.getInputName(req.Input)
	if err != nil {
		return fmt.Errorf("unable to get input video name: %s", err)
	}
	defer finalizeInput()

	outputName, finalizeStream, err := a.getOutputStreamName(req.Output)
	if err != nil {
		return fmt.Errorf("unable to get video filename: %s", err)
	}
	defer func() {
		if finalizeErr := finalizeStream(); finalizeErr != nil {
			logger.Error("unable to finalize stream: %s", finalizeErr)
		}
	}()

	cmd := exec.Command("ffmpeg", "-i", inputName, "-codec:v", "libx264", "-preset", "superfast", "-codec:a", "aac", "-b:a", "128k", "-ac", "2", "-y", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", "-threads", "2", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("unable to generate stream video: %s\n%s", err, buffer.Bytes())

		if cleanErr := a.cleanLocalStream(outputName); cleanErr != nil {
			err = fmt.Errorf("unable to remove generated files: %s: %w", cleanErr, err)
		}

		return err
	}

	log.Info("Generation succeeded!")
	return nil
}

func (a App) isValidStreamName(streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := a.storageApp.Info(streamName)
	if shouldExist {
		if err != nil || info.IsDir {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}

func (a App) getOutputStreamName(name string) (localName string, onEnd func() error, err error) {
	onEnd = func() error {
		return nil
	}

	switch a.storageApp.Name() {
	case filesystem.Name:
		localName = a.storageApp.Path(name)
		return

	case s3.Name:
		localName = filepath.Join(a.tmpFolder, path.Base(name))
		onEnd = a.finalizeStreamForS3(localName, name)
		return

	default:
		err = fmt.Errorf("unknown storage app")
		return
	}
}

func (a App) finalizeStreamForS3(localName, destName string) func() error {
	return func() error {
		if err := a.copyAndCloseLocalFile(localName, destName); err != nil {
			return fmt.Errorf("unable to copy manifest to `%s`: %s", destName, err)
		}

		baseHlsName := strings.TrimSuffix(localName, hlsExtension)
		segments, err := filepath.Glob(fmt.Sprintf("%s*.ts", baseHlsName))
		if err != nil {
			return fmt.Errorf("unable to list hls segments for `%s`: %s", baseHlsName, err)
		}

		outputDir := path.Dir(destName)

		for _, file := range segments {
			segmentName := path.Join(outputDir, filepath.Base(file))
			if err = a.copyAndCloseLocalFile(file, segmentName); err != nil {
				return fmt.Errorf("unable to copy segment to `%s`: %s", segmentName, err)
			}
		}

		if cleanErr := a.cleanLocalStream(localName); cleanErr != nil {
			return fmt.Errorf("unable to clean stream for `%s`: %s", localName, err)
		}

		return nil
	}
}

func (a App) cleanLocalStream(name string) error {
	return a.cleanStream(name, os.Remove, filepath.Glob, "*.ts")
}
