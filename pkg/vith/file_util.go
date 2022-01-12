package vith

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func (a App) readFile(name string) ([]byte, error) {
	reader, err := a.storageApp.ReaderFrom(name)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %s", err)
	}
	defer closeWithLog(reader, "readFile", name)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to read content: %s", err)
	}

	return content, nil
}

func (a App) writeFile(name string, content []byte) error {
	writer, err := a.storageApp.WriterTo(name)
	if err != nil {
		return fmt.Errorf("unable to open writer to storage: %s", err)
	}
	defer closeWithLog(writer, "writeFile", name)

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("unable to write content: %s", err)
	}

	return nil
}

func (a App) listFiles(pattern string) ([]string, error) {
	var items []string

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to compile regular expression: %s", err)
	}

	return items, a.storageApp.Walk(path.Dir(pattern), func(item absto.Item) error {
		if re.MatchString(item.Pathname) {
			items = append(items, item.Pathname)
		}

		return nil
	})
}

func (a App) cleanStream(name string, remove func(string) error, list func(string) ([]string, error), suffix string) error {
	if err := remove(name); err != nil {
		return fmt.Errorf("unable to remove `%s`: %s", name, err)
	}

	rawName := strings.TrimSuffix(name, hlsExtension)

	segments, err := list(rawName + suffix)
	if err != nil {
		return fmt.Errorf("unable to list hls segments for `%s`: %s", rawName, err)
	}

	for _, file := range segments {
		if err := remove(file); err != nil {
			return fmt.Errorf("unable to remove `%s`: %s", file, err)
		}
	}

	return nil
}
