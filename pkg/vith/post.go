package vith

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		a.increaseMetric("http", "thumbnail", "", "invalid")
		return
	}

	if itemType == model.TypePDF {
		if err := a.streamPdf(r.Body, w, r.ContentLength); err != nil {
			httperror.InternalServerError(w, err)
			a.increaseMetric("http", "thumbnail", itemType.String(), "error")
			return
		}

		a.increaseMetric("http", "thumbnail", itemType.String(), "success")

		return
	}

	name := sha.New(time.Now())

	inputName := path.Join(a.tmpFolder, fmt.Sprintf("input_%s", name))
	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.webp", name))

	writer, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "file_error")
		return
	}

	defer cleanFile(inputName)
	defer closeWithLog(writer, "vith.handlePost", inputName)

	if err := loadFile(writer, r); err != nil {
		httperror.InternalServerError(w, err)
		a.increaseMetric("http", "thumbnail", itemType.String(), "load_error")
		return
	}

	a.httpThumbnail(w, model.NewRequest(inputName, outputName, itemType))
}

func loadFile(writer io.Writer, r *http.Request) (err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			} else {
				err = fmt.Errorf("unable to close: %s", err)
			}
		}
	}()

	_, err = io.Copy(writer, r.Body)
	return
}
