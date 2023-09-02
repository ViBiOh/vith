package vith

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) handlePatch(w http.ResponseWriter, r *http.Request) {
	if !s.storage.Enabled() {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	itemType, err := model.ParseItemType(r.URL.Query().Get("type"))
	if err != nil {
		httperror.BadRequest(w, err)
		return
	}

	ctx := r.Context()

	if itemType != model.TypeVideo {
		httperror.BadRequest(w, errors.New("rename is possible for video type only"))
		return
	}

	destinationName := r.URL.Query().Get("to")

	if err := s.isValidStreamName(ctx, r.URL.Path, true); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid source name: %w", err))
		return
	}

	if err := s.isValidStreamName(ctx, destinationName, false); err != nil {
		httperror.BadRequest(w, fmt.Errorf("invalid destination name: %w", err))
		return
	}

	if err := s.renameStream(ctx, r.URL.Path, destinationName); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s Service) renameStream(ctx context.Context, source, destination string) error {
	rawSourceName := strings.TrimSuffix(source, hlsExtension)
	rawDestinationName := strings.TrimSuffix(destination, hlsExtension)

	baseSourceName := path.Base(rawSourceName)
	baseDestinationName := path.Base(rawDestinationName)

	content, err := s.readFile(ctx, source)
	if err != nil {
		return fmt.Errorf("read manifest `%s`: %w", source, err)
	}

	segments, err := s.listFiles(ctx, rawSourceName+`.*\.ts`)
	if err != nil {
		return fmt.Errorf("list hls segments for `%s`: %w", rawSourceName, err)
	}

	if err := s.writeFile(ctx, destination, bytes.ReplaceAll(content, []byte(baseSourceName), []byte(baseDestinationName))); err != nil {
		return fmt.Errorf("write destination file `%s`: %w", destination, err)
	}

	for _, file := range segments {
		newName := rawDestinationName + strings.TrimPrefix(file, rawSourceName)
		if err := s.storage.Rename(ctx, file, newName); err != nil {
			return fmt.Errorf("rename `%s` to `%s`: %w", file, newName, err)
		}
	}

	if err := s.storage.RemoveAll(ctx, source); err != nil {
		return fmt.Errorf("delete `%s`: %w", source, err)
	}

	return nil
}
