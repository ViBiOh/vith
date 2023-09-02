package vith

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

const defaultScale uint64 = 150

func (s Service) handlePost(w http.ResponseWriter, r *http.Request) {
	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		s.increaseMetric(r.Context(), "http", "thumbnail", "", "invalid")
		return
	}

	scale := defaultScale
	if rawScale := r.URL.Query().Get("scale"); len(rawScale) > 0 {
		scale, err = strconv.ParseUint(r.URL.Query().Get("scale"), 10, 64)
		if err != nil {
			httperror.BadRequest(w, fmt.Errorf("parse scale: %w", err))
			s.increaseMetric(r.Context(), "http", "thumbnail", "", "invalid")
			return
		}
	}

	switch itemType {
	case model.TypePDF:
		err = s.pdfThumbnail(r.Context(), r.Body, w, r.ContentLength, scale)

	case model.TypeImage, model.TypeVideo:
		var inputName string
		inputName, err = s.saveFileLocally(r.Body, time.Now().String())
		defer cleanLocalFile(inputName)

		if err == nil {
			outputName := s.getLocalFilename(fmt.Sprintf("output_%s", inputName))
			defer cleanLocalFile(outputName)

			if err = s.getThumbnailGenerator(itemType)(r.Context(), inputName, outputName, scale); err == nil {
				err = copyLocalFile(outputName, w)
			}
		}

	default:
		httperror.BadRequest(w, errors.New("unhandled item type"))
		return
	}

	if err != nil {
		httperror.InternalServerError(w, err)
		s.increaseMetric(r.Context(), "http", "thumbnail", itemType.String(), "error")
		return
	}

	s.increaseMetric(r.Context(), "http", "thumbnail", itemType.String(), "success")
}
