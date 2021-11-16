package vith

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)

	info, err := os.Stat(inputName)
	if err != nil || info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("input `%s` doesn't exist or is a directory", inputName))
		return
	}

	if itemType == model.TypePDF {
		reader, err := os.OpenFile(inputName, os.O_RDONLY, 0o600)
		if err != nil {
			httperror.InternalServerError(w, fmt.Errorf("unable to open input file: %s", err))
		}

		if err := a.streamPdf(reader, w, info.Size()); err != nil {
			httperror.InternalServerError(w, err)
		}

		return
	}

	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.webp", sha.New(time.Now())))
	httpThumbnail(w, model.NewRequest(inputName, outputName, itemType))
}
