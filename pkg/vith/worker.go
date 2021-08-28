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

// Start worker
func (a App) Start() {
	defer close(a.streamRequestQueue)

	if len(a.workingDir) == 0 {
		return
	}

	for req := range a.streamRequestQueue {
		a.generateStream(req)
	}
}

func (a App) generateStream(req streamRequest) {
	outputName := filepath.Join(req.output, fmt.Sprintf("%s%s", filepath.Base(req.input), hlsExtension))

	cmd := exec.Command("ffmpeg", "-i", req.input, "-c:v", "libx264", "-preset", "superfast", "-c:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err := cmd.Run()
	if err != nil {
		logger.Error("unable to generate hls video: %s\n%s", err, buffer.Bytes())

		if err := a.cleanStream(outputName); err != nil {
			logger.Error("unable to remove generated files: %s", err)
		}

		return
	}
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
