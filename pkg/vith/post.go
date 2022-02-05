package vith

import (
	"errors"
	"net/http"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		a.increaseMetric("http", "thumbnail", "", "invalid")
		return
	}

	switch itemType {
	case model.TypePDF:
		err = a.pdfThumbnail(r.Body, w, r.ContentLength)

	case model.TypeImage:
		err = a.streamImageThumbnail(r.Body, w)

	case model.TypeVideo:
		var inputName string
		inputName, err = a.saveFileLocally(r.Body, time.Now().String())
		defer cleanLocalFile(inputName)

		if err == nil {
			err = a.streamVideoThumbnail(inputName, w)
		}

	default:
		httperror.BadRequest(w, errors.New("unhandled item type"))
		return
	}

	if err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "error")
		return
	}

	a.increaseMetric("http", "thumbnail", itemType.String(), "success")
}
