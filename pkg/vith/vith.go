package vith

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

// App of package
type App struct {
	done               chan struct{}
	stop               chan struct{}
	streamRequestQueue chan Request
	tmpFolder          string
	workingDir         string
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
		workingDir: flags.New(prefix, "vith", "WorkDir").Default("", overrides).Label("Working directory for direct access requests").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	return App{
		tmpFolder:          *config.tmpFolder,
		workingDir:         *config.workingDir,
		streamRequestQueue: make(chan Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
	}
}

// Handler for request. Should be use with net/http
func (a App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			a.handleHead(w, r)
		case http.MethodGet:
			a.handleGet(w, r)
		case http.MethodPost:
			a.handlePost(w, r)
		case http.MethodPut:
			a.handlePut(w, r)
		case http.MethodPatch:
			a.handlePatch(w, r)
		case http.MethodDelete:
			a.handleDelete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func (a App) hasDirectAccess() bool {
	return len(a.workingDir) != 0
}

func isValidStreamName(streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if strings.Contains(streamName, "..") {
		return errors.New("path with dots are not allowed")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := os.Stat(streamName)
	if shouldExist {
		if err != nil || info.IsDir() {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}

func generateThumbnail(inputName, outputName string, video bool) error {
	var ffmpegOpts []string
	var customOpts []string

	if duration, err := getContainerDuration(inputName); err != nil {
		logger.Error("unable to get container duration: %s", err)
		if video {
			ffmpegOpts = append(ffmpegOpts, "-ss", "1.000")
		}
		customOpts = []string{
			"-frames:v",
			"1",
		}
	} else {
		ffmpegOpts = append(ffmpegOpts, "-ss", fmt.Sprintf("%.3f", duration/2), "-t", "5")
		customOpts = []string{
			"-vsync",
			"0",
			"-loop",
			"0",
		}
	}

	var animatedOptions string
	if video {
		animatedOptions = ",fps=10"
	}

	ffmpegOpts = append(ffmpegOpts, "-i", inputName, "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150"+animatedOptions, "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture")
	ffmpegOpts = append(ffmpegOpts, customOpts...)
	ffmpegOpts = append(ffmpegOpts, outputName)
	cmd := exec.Command("ffmpeg", ffmpegOpts...)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s: %s", buffer.String(), err)
	}

	return nil
}

func cleanFile(name string) {
	if err := os.Remove(name); err != nil {
		logger.Warn("unable to remove file %s: %s", name, err)
	}
}

func answerThumbnail(w http.ResponseWriter, inputName, outputName string, video bool) {
	defer cleanFile(outputName)

	if err := generateThumbnail(inputName, outputName, video); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	thumbnail, err := os.OpenFile(outputName, os.O_RDONLY, 0o600)
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
	if _, err := io.Copy(w, thumbnail); err != nil {
		logger.Error("unable to copy file: %s", err)
	}
}
