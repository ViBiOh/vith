package vith

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handlePut(w http.ResponseWriter, r *http.Request) {
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
		httperror.BadRequest(w, errors.New("stream are possible for video type only"))
		return
	}

	output := r.URL.Query().Get("output")
	if len(output) == 0 {
		httperror.BadRequest(w, errors.New("output query param is mandatory"))
		return
	}

	logger.WithField("input", r.URL.Path).Info("Adding stream generation in the work queue")

	select {
	case a.streamRequestQueue <- model.NewRequest(r.URL.Path, output, itemType, defaultScale):
		w.WriteHeader(http.StatusAccepted)
	case <-a.stop:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}
