package vith

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) handleHead(w http.ResponseWriter, r *http.Request) {
	if !s.storage.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	if itemType != model.TypeVideo {
		httperror.BadRequest(ctx, w, errors.New("bitrate is possible for video type only"))
		return
	}

	inputName, finalizeInput, err := s.getInputName(ctx, r.URL.Path)
	if err != nil {
		httperror.InternalServerError(ctx, w, fmt.Errorf("get input name: %w", err))
		return
	}

	defer finalizeInput()

	bitrate, duration, err := s.getVideoDetails(ctx, inputName)
	if err != nil {
		httperror.InternalServerError(ctx, w, fmt.Errorf("get bitrate: %w", err))
		return
	}

	w.Header().Set("X-Vith-Bitrate", fmt.Sprintf("%d", bitrate))
	w.Header().Set("X-Vith-Duration", fmt.Sprintf("%.3f", duration))

	w.WriteHeader(http.StatusNoContent)
}

func (s Service) getVideoDetails(ctx context.Context, inputName string) (bireate int64, duration float64, err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "ffprobe")
	defer end(&err)

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=bit_rate:format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err = cmd.Run(); err != nil {
		return 0, 0.0, fmt.Errorf("ffprobe error `%s`: %s", err, buffer.String())
	}

	return parseFfprobeOutput(buffer.String())
}

func parseFfprobeOutput(raw string) (bitrate int64, duration float64, err error) {
	for _, output := range strings.Split(strings.Trim(raw, "\n"), "\n") {
		if bitrate == 0 {
			bitrate, err = strconv.ParseInt(output, 10, 64)
			if err != nil {
				if duration != 0 {
					err = fmt.Errorf("parse bitrate `%s`: %w", output, err)
					return
				}
			} else {
				continue
			}
		}

		if duration == 0 {
			duration, err = strconv.ParseFloat(output, 64)
			if err != nil && bitrate != 0 {
				err = fmt.Errorf("parse duration `%s`: %w", output, err)
				return
			}
		}
	}

	return
}
