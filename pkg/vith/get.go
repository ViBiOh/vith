package vith

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) handleGet(w http.ResponseWriter, r *http.Request) {
	name := sha(time.Now())

	if strings.Contains(r.URL.Path, "..") {
		httperror.BadRequest(w, errors.New("path with dots are not allowed"))
		return
	}

	inputName := filepath.Join(a.workingDir, r.URL.Path)
	outputName := path.Join(a.tmpFolder, fmt.Sprintf("output_%s.jpeg", name))

	cmd := exec.Command("ffmpeg", "-i", inputName, "-ss", "00:00:01", "-frames:v", "1", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err := cmd.Run()

	defer cleanFile(outputName)

	if err != nil {
		httperror.InternalServerError(w, err)
		logger.Error("%s", buffer.String())
		return
	}

	answerFile(w, outputName)
}
