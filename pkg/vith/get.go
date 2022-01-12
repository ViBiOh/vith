package vith

import (
	"fmt"
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

	switch itemType {
	case model.TypeVideo:
		var inputName string
		var finalizeInput func()
		inputName, finalizeInput, err = a.getInputVideoName(r.URL.Path)
		if err != nil {
			err = fmt.Errorf("unable to get input video name: %s", err)
		} else {
			defer finalizeInput()
			err = a.streamVideoThumbnail(inputName, w)
		}

	default:
		err = a.streamThumbnail(r.URL.Path, w, itemType)
	}

	if err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "error")
		return
	}

	a.increaseMetric("http", "thumbnail", itemType.String(), "success")
}
