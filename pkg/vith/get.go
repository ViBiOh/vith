package vith

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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

	scale := defaultScale
	if rawScale := r.URL.Query().Get("scale"); len(rawScale) > 0 {
		scale, err = strconv.ParseUint(r.URL.Query().Get("scale"), 10, 64)
		if err != nil {
			httperror.BadRequest(w, fmt.Errorf("unable to parse scale: %s", err))
			a.increaseMetric("http", "thumbnail", "", "invalid")
			return
		}
	}

	if err := a.storageThumbnail(r.Context(), itemType, r.URL.Path, output, scale); err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
	a.increaseMetric("http", "thumbnail", itemType.String(), "success")
}
