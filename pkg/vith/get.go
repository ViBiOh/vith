package vith

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) handleGet(w http.ResponseWriter, r *http.Request) {
	if !s.storage.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(ctx, w, err)
		s.increaseMetric(r.Context(), "http", "thumbnail", "", "invalid")
		return
	}

	output := r.URL.Query().Get("output")
	if len(output) == 0 {
		httperror.BadRequest(ctx, w, errors.New("output query param is mandatory"))
		s.increaseMetric(r.Context(), "http", "thumbnail", itemType.String(), "invalid")
		return
	}

	scale := defaultScale
	if rawScale := r.URL.Query().Get("scale"); len(rawScale) > 0 {
		scale, err = strconv.ParseUint(r.URL.Query().Get("scale"), 10, 64)
		if err != nil {
			httperror.BadRequest(ctx, w, fmt.Errorf("parse scale: %w", err))
			s.increaseMetric(r.Context(), "http", "thumbnail", "", "invalid")
			return
		}
	}

	if err := s.storageThumbnail(r.Context(), itemType, r.URL.Path, output, scale); err != nil {
		httperror.InternalServerError(ctx, w, err)
		s.increaseMetric(r.Context(), "http", "thumbnail", itemType.String(), "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
	s.increaseMetric(r.Context(), "http", "thumbnail", itemType.String(), "success")
}
