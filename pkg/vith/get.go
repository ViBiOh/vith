package vith

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if !a.storageApp.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		a.increaseMetric("http", "thumbnail", "", "invalid")
		return
	}

	output := r.URL.Query().Get("output")
	if len(output) == 0 {
		httperror.BadRequest(w, errors.New("output query param is mandatory"))
		a.increaseMetric("http", "thumbnail", itemType.String(), "invalid")
		return
	}

	if err := a.storageThumbnail(r.Context(), itemType, r.URL.Path, output); err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
	a.increaseMetric("http", "thumbnail", itemType.String(), "success")
}
