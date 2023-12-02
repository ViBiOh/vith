package vith

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) handlePut(w http.ResponseWriter, r *http.Request) {
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
		httperror.BadRequest(ctx, w, errors.New("stream are possible for video type only"))
		return
	}

	output := r.URL.Query().Get("output")
	if len(output) == 0 {
		httperror.BadRequest(ctx, w, errors.New("output query param is mandatory"))
		return
	}

	slog.InfoContext(ctx, "Adding stream generation in the work queue", "input", r.URL.Path)

	select {
	case s.streamRequestQueue <- model.NewRequest(r.URL.Path, output, itemType, defaultScale):
		w.WriteHeader(http.StatusAccepted)
	case <-s.stop:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}
