package vith

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handleHead(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if itemType != model.TypeVideo {
		httperror.BadRequest(w, errors.New("bitrate is possible for video type only"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)

	bitrate, err := getVideoBitrate(inputName)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to get bitrate: %s", err))
	}

	duration, err := getContainerDuration(inputName)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to get duration: %s", err))
	}

	w.Header().Set("X-Vith-Bitrate", fmt.Sprintf("%d", bitrate))
	w.Header().Set("X-Vith-Duration", fmt.Sprintf("%.3f", duration))

	w.WriteHeader(http.StatusOK)
}

func getVideoBitrate(name string) (int64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=bit_rate", "-of", "default=noprint_wrappers=1:nokey=1", name)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return 0.0, fmt.Errorf("ffmpeg error `%s`: %s", err, buffer.String())
	}

	output := strings.Trim(buffer.String(), "\n")

	duration, err := strconv.ParseInt(output, 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf("unable to parse bitrate `%s`: %s", output, err)
	}

	return duration, nil
}

func getContainerDuration(name string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", name)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return 0.0, fmt.Errorf("ffmpeg error `%s`: %s", err, buffer.String())
	}

	output := strings.Trim(buffer.String(), "\n")

	duration, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return 0.0, fmt.Errorf("unable to parse duration `%s`: %s", output, err)
	}

	return duration, nil
}
