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
	"os/exec"
	"strconv"
	"strings"
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

func answerThumbnail(w http.ResponseWriter, inputName, outputName string) {
	duration, err := getContainerDuration(inputName)
	if err != nil {
		logger.Error("unable to get container duration: %s", err)
		duration = 02 // so we take the first second
	}

	cmd := exec.Command("ffmpeg", "-ss", fmt.Sprintf("%.3f", duration/2), "-i", inputName, "-frames:v", "1", outputName)

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

	thumbnail, err := os.OpenFile(outputName, os.O_RDONLY, 0600)
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

	w.WriteHeader(http.StatusOK)
	if _, err := io.CopyBuffer(w, thumbnail, buffer.Bytes()); err != nil {
		logger.Error("unable to copy file: %s", err)
	}
}

func getContainerDuration(name string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", name)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return 0.0, fmt.Errorf("ffmpeg error `%s`: %s", err, buffer.String())
	}

	output := strings.Trim(buffer.String(), "\n")

	duration, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return 0.0, fmt.Errorf("unable to parse duration `%s`: %s", output, err)
	}

	return duration, nil
}

func cleanFile(name string) {
	if err := os.Remove(name); err != nil {
		logger.Warn("unable to remove file %s: %s", name, err)
	}
}

func sha(o interface{}) string {
	hasher := sha1.New()

	// no err check https://golang.org/pkg/hash/#Hash
	_, _ = hasher.Write([]byte(fmt.Sprintf("%#v", o)))

	return hex.EncodeToString(hasher.Sum(nil))
}
