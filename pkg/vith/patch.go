package vith

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
)

func (a App) handlePatch(w http.ResponseWriter, r *http.Request) {
	if !a.hasDirectAccess() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sourceName := filepath.Join(a.workingDir, r.URL.Path)
	destinationName := filepath.Join(a.workingDir, r.URL.Query().Get("to"))

	if err := isValidStreamName(sourceName, true); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid source name: %s", err))
		return
	}

	if err := isValidStreamName(destinationName, false); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid destination name: %s", err))
		return
	}

	if err := a.renameStream(sourceName, destinationName); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a App) renameStream(source, destination string) error {
	rawSourceName := strings.TrimSuffix(source, hlsExtension)
	rawDestinationName := strings.TrimSuffix(destination, hlsExtension)

	baseSourceName := filepath.Base(rawSourceName)
	baseDestinationName := filepath.Base(rawDestinationName)

	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("unable to read source file `%s`: %s", source, err)
	}

	segments, err := filepath.Glob(rawSourceName + "*.ts")
	if err != nil {
		return fmt.Errorf("unable to list hls segments for `%s`: %s", rawSourceName, err)
	}

	if err := os.WriteFile(destination, bytes.ReplaceAll(content, []byte(baseSourceName), []byte(baseDestinationName)), 0600); err != nil {
		return fmt.Errorf("unable to write destination file `%s`: %s", destination, err)
	}

	for _, file := range segments {
		newName := rawDestinationName + strings.TrimPrefix(file, rawSourceName)
		if err := os.Rename(file, newName); err != nil {
			return fmt.Errorf("unable to rename `%s` to `%s`: %s", file, newName, err)
		}
	}

	if err := os.Remove(source); err != nil {
		return fmt.Errorf("unable to delete `%s`: %s", source, err)
	}

	return nil
}
