package vith

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

const (
	hlsExtension = ".m3u8"
)

type streamRequest struct {
	input  string
	output string
}

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
		a.generateStream(req)
	}
}

func (a App) stopOnce() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
}

func (a App) generateStream(req streamRequest) {
	log := logger.WithField("input", req.input).WithField("output", req.output)
	log.Info("Generating stream...")

	inputFilename := filepath.Base(req.input)
	inputFilename = strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename))

	outputName := filepath.Join(req.output, fmt.Sprintf("%s%s", inputFilename, hlsExtension))

	cmd := exec.Command("ffmpeg", "-i", req.input, "-codec:v", "libx264", "-preset", "superfast", "-codec:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err := cmd.Run()
	if err != nil {
		log.Error("unable to generate hls video: %s\n%s", err, buffer.Bytes())

		if err := a.cleanStream(outputName); err != nil {
			log.Error("unable to remove generated files: %s", err)
		}

		return
	}

	log.Info("Generation succeeded!")
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
