package vith

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	if !a.hasDirectAccess() {
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
		if err := a.generateStream(req); err != nil {
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

func (a App) generateStream(req model.Request) error {
	log := logger.WithField("input", req.Input).WithField("output", req.Output)
	log.Info("Generating stream...")

	cmd := exec.Command("ffmpeg", "-i", req.Input, "-codec:v", "libx264", "-preset", "superfast", "-codec:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", "-threads", "2", req.Output)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("unable to generate stream video: %s\n%s", err, buffer.Bytes())

		if cleanErr := a.cleanStream(req.Output); cleanErr != nil {
			err = fmt.Errorf("unable to remove generated files: %s: %w", cleanErr, err)
		}

		return err
	}

	log.Info("Generation succeeded!")
	return nil
}

func (a App) cleanStream(outputName string) error {
	if err := os.Remove(outputName); err != nil {
		return fmt.Errorf("unable to remove `%s`: %s", outputName, err)
	}

	rawName := strings.TrimSuffix(outputName, hlsExtension)

	segments, err := filepath.Glob(fmt.Sprintf("%s*.ts", rawName))
	if err != nil {
		return fmt.Errorf("unable to list hls segments for `%s`: %s", rawName, err)
	}

	for _, file := range segments {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("unable to remove `%s`: %s", file, err)
		}
	}

	return nil
}

func isValidStreamName(streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if strings.Contains(streamName, "..") {
		return errors.New("path with dots are not allowed")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := os.Stat(streamName)
	if shouldExist {
		if err != nil || info.IsDir() {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}
