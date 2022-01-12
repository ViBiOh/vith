package vith

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

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
		a.increaseMetric("http", "thumbnail", "", "invalid")
		return
	}

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		a.increaseMetric("http", "thumbnail", itemType.String(), "input_invalid")
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)

	if itemType != model.TypeVideo {
		if err := a.fileThumbnail(inputName, w, "http", itemType); err != nil {
			httperror.InternalServerError(w, err)
		}
		return
	}

	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.webp", sha.New(inputName)))
	a.httpVideoThumbnail(w, model.NewRequest(inputName, outputName, itemType))
}
