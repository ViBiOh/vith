package vith

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

// App of package
type App interface {
	Handler() http.Handler
}

// Handler for request. Should be use with net/http
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := sha(time.Now())
		inputName := fmt.Sprintf("/tmp/input_%s", name)
		outputName := fmt.Sprintf("/tmp/output_%s.jpeg", name)
		copyBuffer := make([]byte, 32*1024)

		inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if _, err := io.CopyBuffer(inputFile, r.Body, copyBuffer); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		cmd := exec.Command("ffmpeg", "-i", inputName, "-vf", "thumbnail", "-frames:v", "1", outputName)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			httperror.InternalServerError(w, err)
			logger.Error("%s", out.String())
			return
		}

		thumbnail, err := os.OpenFile(outputName, os.O_RDONLY, 0600)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := io.CopyBuffer(w, thumbnail, copyBuffer); err != nil {
			logger.Error("%s", err)
		}

		if err := os.Remove(outputName); err != nil {
			logger.Error("%s", err)
		}
	})
}

func sha(o interface{}) string {
	hasher := sha1.New()

	// no err check https://golang.org/pkg/hash/#Hash
	if _, err := hasher.Write([]byte(fmt.Sprintf("%#v", o))); err != nil {
		logger.Error("%s", err)
		return ""
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
