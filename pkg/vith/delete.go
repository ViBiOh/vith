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

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if itemType != model.TypeVideo {
		httperror.BadRequest(w, errors.New("deletion is possible for video type only"))
		return
	}

	ctx := r.Context()

	if err := s.isValidStreamName(ctx, r.URL.Path, true); err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if err := s.cleanStream(ctx, r.URL.Path, s.storage.RemoveAll, s.listFiles, `.*\.ts`); err != nil {
		httperror.InternalServerError(w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
