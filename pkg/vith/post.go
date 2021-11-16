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
		return
	}

	if itemType == model.TypePDF {
		if err := a.streamPdf(r.Body, w, r.ContentLength); err != nil {
			httperror.InternalServerError(w, err)
		}

		return
	}

	name := sha.New(time.Now())

	inputName := path.Join(a.tmpFolder, fmt.Sprintf("input_%s", name))
	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.webp", name))

	inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer cleanFile(inputName)

	if err := loadFile(inputFile, r); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	httpThumbnail(w, model.NewRequest(inputName, outputName, itemType))
}

func loadFile(writer io.WriteCloser, r *http.Request) (err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}

		if closeErr := writer.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}
	}()

	_, err = io.Copy(writer, r.Body)
	return
}
