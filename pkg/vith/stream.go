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

func (s Service) Done() <-chan struct{} {
	return s.done
}

func (s Service) Start(ctx context.Context) {
	defer close(s.done)
	defer close(s.streamRequestQueue)
	defer s.stopOnce()

	if !s.storage.Enabled() {
		return
	}

	done := ctx.Done()

	go func() {
		defer s.stopOnce()

		select {
		case <-done:
		case <-s.done:
		}
	}()

	for req := range s.streamRequestQueue {
		if err := s.generateStream(context.Background(), req); err != nil {
			slog.Error("generate stream", "err", err)
		}
	}
}

func (s Service) stopOnce() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s Service) generateStream(ctx context.Context, req model.Request) error {
	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "stream")
	defer end(&err)

	log := slog.With("input", req.Input).With("output", req.Output)
	log.Info("Generating stream...")

	inputName, finalizeInput, err := s.getInputName(ctx, req.Input)
	if err != nil {
		return fmt.Errorf("get input video name: %w", err)
	}
	defer finalizeInput()

	outputName, finalizeStream, err := s.getOutputStreamName(ctx, req.Output)
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

		if cleanErr := s.cleanLocalStream(ctx, outputName); cleanErr != nil {
			err = fmt.Errorf("remove generated files: %s: %w", cleanErr, err)
		}

		return err
	}

	log.Info("Generation succeeded!")
	return nil
}

func (s Service) isValidStreamName(ctx context.Context, streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := s.storage.Stat(ctx, streamName)
	if shouldExist {
		if err != nil || info.IsDir() {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}

func (s Service) getOutputStreamName(ctx context.Context, name string) (localName string, onEnd func() error, err error) {
	onEnd = func() error {
		return nil
	}

	switch s.storage.Name() {
	case filesystem.Name:
		localName = s.storage.Path(name)
		return

	case s3.Name:
		localName = filepath.Join(s.tmpFolder, path.Base(name))
		onEnd = s.finalizeStreamForS3(ctx, localName, name)
		return

	default:
		err = fmt.Errorf("unknown storage app")
		return
	}
}

func (s Service) finalizeStreamForS3(ctx context.Context, localName, destName string) func() error {
	return func() error {
		if err := s.copyAndCloseLocalFile(ctx, localName, destName); err != nil {
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
			if err = s.copyAndCloseLocalFile(ctx, file, segmentName); err != nil {
				return fmt.Errorf("copy segment to `%s`: %w", segmentName, err)
			}
		}

		if cleanErr := s.cleanLocalStream(ctx, localName); cleanErr != nil {
			return fmt.Errorf("clean stream for `%s`: %w", localName, err)
		}

		return nil
	}
}

func (s Service) cleanLocalStream(ctx context.Context, name string) error {
	return s.cleanStream(ctx, name, func(_ context.Context, name string) error {
		return os.Remove(name)
	}, func(_ context.Context, name string) ([]string, error) {
		return filepath.Glob(name)
	}, "*.ts")
}
