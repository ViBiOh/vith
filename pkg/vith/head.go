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
	"github.com/ViBiOh/vith/pkg/model"
	"go.opentelemetry.io/otel/trace"
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

	ctx := r.Context()

	inputName, finalizeInput, err := a.getInputName(ctx, r.URL.Path)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to get input name: %s", err))
		return
	}

	defer finalizeInput()

	bitrate, duration, err := a.getVideoDetails(ctx, inputName)
	if err != nil {
		httperror.InternalServerError(w, fmt.Errorf("unable to get bitrate: %s", err))
		return
	}

	w.Header().Set("X-Vith-Bitrate", fmt.Sprintf("%d", bitrate))
	w.Header().Set("X-Vith-Duration", fmt.Sprintf("%.3f", duration))

	w.WriteHeader(http.StatusNoContent)
}

func (a App) getVideoDetails(ctx context.Context, inputName string) (int64, float64, error) {
	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "ffprobe")
		defer span.End()
	}

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=bit_rate:format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
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
