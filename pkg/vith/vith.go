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
	"path"
	"sync"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}
)

// App of package
type App interface {
	Handler() http.Handler
}

// Handler for request. Should be use with net/http
func Handler(tmpFolder string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		name := sha(time.Now())
		inputName := path.Join(tmpFolder, fmt.Sprintf("input_%s", name))
		outputName := path.Join(tmpFolder, fmt.Sprintf("output_%s.jpeg", name))

		inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		defer cleanFile(inputName, inputFile)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		buffer := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buffer)

		if _, err := io.CopyBuffer(inputFile, r.Body, buffer.Bytes()); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		cmd := exec.Command("ffmpeg", "-i", inputName, "-vf", "thumbnail", "-frames:v", "1", outputName)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err = cmd.Run()
		defer cleanFile(outputName, nil)

		if err != nil {
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
		if _, err := io.CopyBuffer(w, thumbnail, buffer.Bytes()); err != nil {
			logger.Error("%s", err)
		}
	})
}

func cleanFile(name string, file *os.File) {
	if file != nil {
		if err := file.Close(); err != nil {
			logger.Warn("unable to close file %s: %s", name, err)
		}
	}

	if err := os.Remove(name); err != nil {
		logger.Warn("unable to remove file %s: %s", name, err)
	}
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
