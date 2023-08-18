package vith

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/model"
)

// Done close when work is over
func (a App) Done() <-chan struct{} {
	return a.done
}

// Start worker
func (a App) Start(ctx context.Context) {
	defer close(a.done)
	defer close(a.streamRequestQueue)
	defer a.stopOnce()

	if !a.storageApp.Enabled() {
		return
	}

	done := ctx.Done()

	go func() {
		defer a.stopOnce()

		select {
		case <-done:
		case <-a.done:
		}
	}()

	for req := range a.streamRequestQueue {
		if err := a.generateStream(context.Background(), req); err != nil {
			slog.Error("generate stream", "err", err)
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
	var err error

	ctx, end := telemetry.StartSpan(ctx, a.tracer, "stream")
	defer end(&err)

	log := slog.With("input", req.Input).With("output", req.Output)
	log.Info("Generating stream...")

	inputName, finalizeInput, err := a.getInputName(ctx, req.Input)
	if err != nil {
		return fmt.Errorf("get input video name: %w", err)
	}
	defer finalizeInput()

	outputName, finalizeStream, err := a.getOutputStreamName(ctx, req.Output)
	if err != nil {
		return fmt.Errorf("get video filename: %w", err)
	}
	defer func() {
		if finalizeErr := finalizeStream(); finalizeErr != nil {
			slog.Error("finalize stream", "err", finalizeErr)
		}
	}()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-hwaccel", "auto", "-i", inputName, "-codec:v", "libx264", "-preset", "superfast", "-codec:a", "aac", "-b:a", "128k", "-ac", "2", "-y", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", "-threads", "2", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("generate stream video: %s\n%s", err, buffer.Bytes())

		if cleanErr := a.cleanLocalStream(ctx, outputName); cleanErr != nil {
			err = fmt.Errorf("remove generated files: %s: %w", cleanErr, err)
		}

		return err
	}

	log.Info("Generation succeeded!")
	return nil
}

func (a App) isValidStreamName(ctx context.Context, streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := a.storageApp.Stat(ctx, streamName)
	if shouldExist {
		if err != nil || info.IsDir() {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}

func (a App) getOutputStreamName(ctx context.Context, name string) (localName string, onEnd func() error, err error) {
	onEnd = func() error {
		return nil
	}

	switch a.storageApp.Name() {
	case filesystem.Name:
		localName = a.storageApp.Path(name)
		return

	case s3.Name:
		localName = filepath.Join(a.tmpFolder, path.Base(name))
		onEnd = a.finalizeStreamForS3(ctx, localName, name)
		return

	default:
		err = fmt.Errorf("unknown storage app")
		return
	}
}

func (a App) finalizeStreamForS3(ctx context.Context, localName, destName string) func() error {
	return func() error {
		if err := a.copyAndCloseLocalFile(ctx, localName, destName); err != nil {
			return fmt.Errorf("copy manifest to `%s`: %w", destName, err)
		}

		baseHlsName := strings.TrimSuffix(localName, hlsExtension)
		segments, err := filepath.Glob(fmt.Sprintf("%s*.ts", baseHlsName))
		if err != nil {
			return fmt.Errorf("list hls segments for `%s`: %w", baseHlsName, err)
		}

		outputDir := path.Dir(destName)

		for _, file := range segments {
			segmentName := path.Join(outputDir, filepath.Base(file))
			if err = a.copyAndCloseLocalFile(ctx, file, segmentName); err != nil {
				return fmt.Errorf("copy segment to `%s`: %w", segmentName, err)
			}
		}

		if cleanErr := a.cleanLocalStream(ctx, localName); cleanErr != nil {
			return fmt.Errorf("clean stream for `%s`: %w", localName, err)
		}

		return nil
	}
}

func (a App) cleanLocalStream(ctx context.Context, name string) error {
	return a.cleanStream(ctx, name, func(_ context.Context, name string) error {
		return os.Remove(name)
	}, func(_ context.Context, name string) ([]string, error) {
		return filepath.Glob(name)
	}, "*.ts")
}
