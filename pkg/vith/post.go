package vith

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) handlePost(w http.ResponseWriter, r *http.Request) {
	name := sha(time.Now())

	inputName := path.Join(a.tmpFolder, fmt.Sprintf("input_%s", name))
	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.jpeg", name))

	inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer cleanFile(inputName)

	if err := loadFile(inputFile, r); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	cmd := exec.Command("ffmpeg", "-i", inputName, "-vf", "thumbnail", "-frames:v", "1", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err = cmd.Run()

	defer cleanFile(outputName)

	if err != nil {
		httperror.InternalServerError(w, err)
		logger.Error("%s", buffer.String())
		return
	}

	answerFile(w, outputName)
}

func loadFile(writer io.WriteCloser, r *http.Request) (err error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}

		if closeErr := writer.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}
	}()

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, r.Body, buffer.Bytes())
	return
}
