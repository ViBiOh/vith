package vith

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) handleDelete(w http.ResponseWriter, r *http.Request) {
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
		httperror.BadRequest(ctx, w, errors.New("deletion is possible for video type only"))
		return
	}

	if err := s.isValidStreamName(ctx, r.URL.Path, true); err != nil {
		httperror.BadRequest(ctx, w, err)
		return
	}

	if err := s.cleanStream(ctx, r.URL.Path, s.storage.RemoveAll, s.listFiles, `.*\.ts`); err != nil {
		httperror.InternalServerError(ctx, w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
