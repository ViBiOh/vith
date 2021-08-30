package vith

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handleDelete(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)

	if err := isValidStreamName(inputName); err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if err := a.cleanStream(inputName); err != nil {
		httperror.InternalServerError(w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
