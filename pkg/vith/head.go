package vith

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handleHead(w http.ResponseWriter, r *http.Request) {
	if !a.storageApp.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	reader, err := a.storageApp.ReadFrom(r.URL.Path)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to read from storage: %s", err))
		return
	}
	defer closeWithLog(reader, "handleHead", r.URL.Path)

	bitrate, duration, err := getVideoDetails(reader)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to get bitrate: %s", err))
		return
	}

	w.Header().Set("X-Vith-Bitrate", fmt.Sprintf("%d", bitrate))
	w.Header().Set("X-Vith-Duration", fmt.Sprintf("%.3f", duration))

	w.WriteHeader(http.StatusNoContent)
}

func getVideoDetails(input io.Reader) (bitrate int64, duration float64, err error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=bit_rate:format=duration", "-of", "default=noprint_wrappers=1:nokey=1", "-")

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdin = input
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err = cmd.Run(); err != nil {
		err = fmt.Errorf("ffmpeg error `%s`: %s", err, buffer.String())
		return
	}

	for _, output := range strings.Split(strings.Trim(buffer.String(), "\n"), "\n") {
		if bitrate == 0 {
			bitrate, err = strconv.ParseInt(output, 10, 64)
			if err != nil {
				if duration != 0 {
					err = fmt.Errorf("unable to parse bitrate `%s`: %s", output, err)
					return
				}
			} else {
				continue
			}
		}

		if duration == 0 {
			duration, err = strconv.ParseFloat(output, 64)
			if err != nil && bitrate != 0 {
				err = fmt.Errorf("unable to parse duration `%s`: %s", output, err)
				return
			}
		}
	}

	return
}
