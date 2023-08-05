package vith

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handleDelete(w http.ResponseWriter, r *http.Request) {
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
		httperror.BadRequest(w, errors.New("deletion is possible for video type only"))
		return
	}

	ctx := r.Context()

	if err := a.isValidStreamName(ctx, r.URL.Path, true); err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if err := a.cleanStream(ctx, r.URL.Path, a.storageApp.RemoveAll, a.listFiles, `.*\.ts`); err != nil {
		httperror.InternalServerError(w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
