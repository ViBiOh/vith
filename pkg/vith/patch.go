package vith

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (a App) handlePatch(w http.ResponseWriter, r *http.Request) {
	if !a.storageApp.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if itemType != model.TypeVideo {
		httperror.BadRequest(w, errors.New("rename is possible for video type only"))
		return
	}

	destinationName := r.URL.Query().Get("to")

	if err := a.isValidStreamName(r.URL.Path, true); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid source name: %s", err))
		return
	}

	if err := a.isValidStreamName(destinationName, false); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid destination name: %s", err))
		return
	}

	if err := a.renameStream(r.URL.Path, destinationName); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a App) renameStream(source, destination string) error {
	rawSourceName := strings.TrimSuffix(source, hlsExtension)
	rawDestinationName := strings.TrimSuffix(destination, hlsExtension)

	baseSourceName := path.Base(rawSourceName)
	baseDestinationName := path.Base(rawDestinationName)

	content, err := a.readFile(source)
	if err != nil {
		return fmt.Errorf("unable to read manifest `%s`: %s", source, err)
	}

	segments, err := a.listFiles(rawSourceName + `.*\.ts`)
	if err != nil {
		return fmt.Errorf("unable to list hls segments for `%s`: %s", rawSourceName, err)
	}

	if err := a.writeFile(destination, bytes.ReplaceAll(content, []byte(baseSourceName), []byte(baseDestinationName))); err != nil {
		return fmt.Errorf("unable to write destination file `%s`: %s", destination, err)
	}

	for _, file := range segments {
		newName := rawDestinationName + strings.TrimPrefix(file, rawSourceName)
		if err := a.storageApp.Rename(file, newName); err != nil {
			return fmt.Errorf("unable to rename `%s` to `%s`: %s", file, newName, err)
		}
	}

	if err := a.storageApp.Remove(source); err != nil {
		return fmt.Errorf("unable to delete `%s`: %s", source, err)
	}

	return nil
}
