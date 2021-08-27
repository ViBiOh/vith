package vith

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

const (
	hlsExtension = ".m3u8"
)

func (a App) handlePut(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	outputFolder := r.URL.Query().Get("output")
	if len(outputFolder) == 0 {
		httperror.BadRequest(w, errors.New("output query param is mandatory"))
		return
	}

	if strings.Contains(outputFolder, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)
	outputName := filepath.Join(a.workingDir, outputFolder)

	if info, err := os.Stat(inputName); err != nil || info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("input `%s` doesn't exist or is a directory", inputName))
		return
	}

	if info, err := os.Stat(outputName); err != nil || !info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("output `%s` doesn't exist or is not a directory", outputName))
		return
	}

	go a.generateHLS(inputName, outputName)
	w.WriteHeader(http.StatusAccepted)
}

func (a App) generateHLS(inputName, outputFolder string) {
	outputName := filepath.Join(outputFolder, fmt.Sprintf("%s%s", filepath.Base(inputName), hlsExtension))

	cmd := exec.Command("ffmpeg", "-i", inputName, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err := cmd.Run()
	if err != nil {
		logger.Error("unable to generate hls video: %s\n%s", err, buffer.Bytes())

		if err := a.cleanHLS(outputName); err != nil {
			logger.Error("unable to remove generated files: %s", err)
		}

		return
	}
}

func (a App) cleanHLS(outputName string) error {
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
