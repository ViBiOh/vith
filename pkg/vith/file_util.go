package vith

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func (a App) readFile(ctx context.Context, name string) ([]byte, error) {
	reader, err := a.storageApp.ReadFrom(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	defer closeWithLog(reader, "readFile", name)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	return content, nil
}

func (a App) writeFile(ctx context.Context, name string, content []byte) error {
	if err := a.storageApp.WriteSizedTo(ctx, name, int64(len(content)), bytes.NewBuffer(content)); err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	return nil
}

func (a App) listFiles(ctx context.Context, pattern string) ([]string, error) {
	var items []string

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regular expression: %w", err)
	}

	return items, a.storageApp.Walk(ctx, path.Dir(pattern), func(item absto.Item) error {
		if re.MatchString(item.Pathname) {
			items = append(items, item.Pathname)
		}

		return nil
	})
}

func (a App) cleanStream(ctx context.Context, name string, remove func(context.Context, string) error, list func(context.Context, string) ([]string, error), suffix string) error {
	if err := remove(ctx, name); err != nil {
		return fmt.Errorf("remove `%s`: %w", name, err)
	}

	rawName := strings.TrimSuffix(name, hlsExtension)

	segments, err := list(ctx, rawName+suffix)
	if err != nil {
		return fmt.Errorf("list hls segments for `%s`: %w", rawName, err)
	}

	for _, file := range segments {
		if err := remove(ctx, file); err != nil {
			return fmt.Errorf("remove `%s`: %w", file, err)
		}
	}

	return nil
}
