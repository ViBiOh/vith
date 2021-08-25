package vith

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)
	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.jpeg", sha(time.Now())))

	answerThumbnail(w, inputName, outputName)
}
