package vith

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handlePut(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
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

	outputFolder := r.URL.Query().Get("output")
	if len(outputFolder) == 0 {
		httperror.BadRequest(w, errors.New("output query param is mandatory"))
		return
	}

	if strings.Contains(outputFolder, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)
	outputName := filepath.Join(a.workingDir, outputFolder)

	if info, err := os.Stat(inputName); err != nil || info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("input `%s` doesn't exist or is a directory", inputName))
		return
	}

	if info, err := os.Stat(outputName); err != nil || !info.IsDir() {
		httperror.BadRequest(w, fmt.Errorf("output `%s` doesn't exist or is not a directory", outputName))
		return
	}

	logger.WithField("input", inputName).Info("Adding stream generation in the work queue")

	select {
	case a.streamRequestQueue <- model.NewRequest(inputName, outputName, itemType):
		w.WriteHeader(http.StatusAccepted)
	case <-a.stop:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}
