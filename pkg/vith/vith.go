package vith

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
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
type App struct {
	tmpFolder  string
	workingDir string
}

// Config of package
type Config struct {
	tmpFolder  *string
	workingDir *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder:  flags.New(prefix, "vith", "TmpFolder").Default("/tmp", overrides).Label("Folder used for temporary files storage").ToString(fs),
		workingDir: flags.New(prefix, "vith", "WorkDir").Default("", overrides).Label("Working directory for GET requests").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	return App{
		tmpFolder:  *config.tmpFolder,
		workingDir: *config.workingDir,
	}
}

// Handler for request. Should be use with net/http
func (a App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			a.handlePost(w, r)
		case http.MethodGet:
			a.handleGet(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func answerFile(w http.ResponseWriter, filename string) {
	thumbnail, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer func() {
		if closeErr := thumbnail.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}
	}()

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	w.WriteHeader(http.StatusOK)
	if _, err := io.CopyBuffer(w, thumbnail, buffer.Bytes()); err != nil {
		logger.Error("unable to copy file: %s", err)
	}
}

func cleanFile(name string) {
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
