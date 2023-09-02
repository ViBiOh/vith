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

func (s Service) readFile(ctx context.Context, name string) ([]byte, error) {
	reader, err := s.storage.ReadFrom(ctx, name)
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

func (s Service) writeFile(ctx context.Context, name string, content []byte) error {
	if err := s.storage.WriteTo(ctx, name, bytes.NewBuffer(content), absto.WriteOpts{Size: int64(len(content))}); err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	return nil
}

func (s Service) listFiles(ctx context.Context, pattern string) ([]string, error) {
	var items []string

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regular expression: %w", err)
	}

	return items, s.storage.Walk(ctx, path.Dir(pattern), func(item absto.Item) error {
		if re.MatchString(item.Pathname) {
			items = append(items, item.Pathname)
		}

		return nil
	})
}

func (s Service) cleanStream(ctx context.Context, name string, remove func(context.Context, string) error, list func(context.Context, string) ([]string, error), suffix string) error {
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
