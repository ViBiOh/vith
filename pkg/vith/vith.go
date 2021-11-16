package vith

import (
	"bytes"
	"context"
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
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
)

const (
	// Width is the width of each thumbnail generated
	Width = 150

	// Height is the width of each thumbnail generated
	Height = 150
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}

	slowClient = request.CreateClient(time.Minute, request.NoRedirection)
)

// App of package
type App struct {
	done               chan struct{}
	stop               chan struct{}
	streamRequestQueue chan model.Request
	tmpFolder          string
	workingDir         string

	imaginaryReq request.Request
}

// Config of package
type Config struct {
	tmpFolder  *string
	workingDir *string

	imaginaryURL  *string
	imaginaryUser *string
	imaginaryPass *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		tmpFolder:  flags.New(prefix, "vith", "TmpFolder").Default("/tmp", overrides).Label("Folder used for temporary files storage").ToString(fs),
		workingDir: flags.New(prefix, "vith", "WorkDir").Default("", overrides).Label("Working directory for direct access requests").ToString(fs),

		imaginaryURL:  flags.New(prefix, "thumbnail", "ImaginaryURL").Default("http://image:9000", nil).Label("Imaginary URL").ToString(fs),
		imaginaryUser: flags.New(prefix, "thumbnail", "ImaginaryUser").Default("", nil).Label("Imaginary Basic Auth User").ToString(fs),
		imaginaryPass: flags.New(prefix, "thumbnail", "ImaginaryPassword").Default("", nil).Label("Imaginary Basic Auth Password").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	imaginaryReq := request.New().WithClient(slowClient).Post(*config.imaginaryURL).BasicAuth(strings.TrimSpace(*config.imaginaryUser), *config.imaginaryPass)
	if !imaginaryReq.IsZero() {
		imaginaryReq = imaginaryReq.Path(fmt.Sprintf("/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=webp", Width, Height))
	}

	return App{
		tmpFolder:          *config.tmpFolder,
		workingDir:         *config.workingDir,
		streamRequestQueue: make(chan model.Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
		imaginaryReq:       imaginaryReq,
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

func httpThumbnail(w http.ResponseWriter, req model.Request) {
	defer cleanFile(req.Output)

	if err := thumbnail(req); err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	reader, err := os.OpenFile(req.Output, os.O_RDONLY, 0o600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, reader); err != nil {
		logger.Error("unable to copy file: %s", err)
	}

	if err := reader.Close(); err != nil {
		logger.Error("unable to close reader for http: %s", err)
	}
}

func thumbnail(req model.Request) error {
	var ffmpegOpts []string
	var customOpts []string

	isVideo := req.ItemType == model.TypeVideo

	if duration, err := getContainerDuration(req.Input); err != nil {
		logger.Error("unable to get container duration: %s", err)
		if isVideo {
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
	if isVideo {
		animatedOptions = ",fps=10"
	}

	ffmpegOpts = append(ffmpegOpts, "-i", req.Input, "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150"+animatedOptions, "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture")
	ffmpegOpts = append(ffmpegOpts, customOpts...)
	ffmpegOpts = append(ffmpegOpts, req.Output)
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

func (a App) pdf(req model.Request) error {
	stats, err := os.Stat(req.Input)
	if err != nil {
		return fmt.Errorf("unable to stats input file: %s", err)
	}

	reader, err := os.OpenFile(req.Input, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open input file: %s", err)
	}

	writer, err := os.OpenFile(req.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		if closeErr := reader.Close(); closeErr != nil {
			logger.Error("unable to close input reader: %s", closeErr)
		}

		return fmt.Errorf("unable to open output file: %s", err)
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			logger.Error("unable to close pdf writer: %s", closeErr)
		}
	}()

	return a.streamPdf(reader, writer, stats.Size())
}

func (a App) streamPdf(reader io.ReadCloser, writer io.Writer, contentLength int64) error {
	r, err := a.imaginaryReq.Build(context.Background(), reader)
	if err != nil {
		if closeErr := reader.Close(); closeErr != nil {
			logger.Error("unable to close reader: %s", err)
		}
		return fmt.Errorf("unable to build request: %s", err)
	}

	r.ContentLength = contentLength

	resp, err := request.DoWithClient(slowClient, r)
	if err != nil {
		if closeErr := reader.Close(); closeErr != nil {
			logger.Error("unable to close reader: %s", err)
		}

		return fmt.Errorf("unable to request imaginary: %s", err)
	}

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("unable to copy imaginary response: %s", err)
	}

	return nil
}
