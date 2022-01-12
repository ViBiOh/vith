package vith

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Width is the width of each thumbnail generated
	Width = 150

	// Height is the width of each thumbnail generated
	Height = 150

	hlsExtension = ".m3u8"
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
	metric             *prometheus.CounterVec
	tmpFolder          string
	workingDir         string
	imaginaryReq       request.Request
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
func New(config Config, prometheusRegisterer prometheus.Registerer) App {
	imaginaryReq := request.Post(*config.imaginaryURL).WithClient(slowClient).BasicAuth(strings.TrimSpace(*config.imaginaryUser), *config.imaginaryPass)
	if !imaginaryReq.IsZero() {
		imaginaryReq = imaginaryReq.Path(fmt.Sprintf("/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=webp", Width, Height))
	}

	return App{
		tmpFolder:          *config.tmpFolder,
		workingDir:         *config.workingDir,
		streamRequestQueue: make(chan model.Request, 4),
		stop:               make(chan struct{}),
		done:               make(chan struct{}),
		metric:             prom.CounterVec(prometheusRegisterer, "vith", "", "item", "source", "kind", "type", "state"),
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

func (a App) httpVideoThumbnail(w http.ResponseWriter, req model.Request) {
	defer cleanFile(req.Output)

	if err := videoThumbnail(req); err != nil {
		a.increaseMetric("http", "thumbnail", req.ItemType.String(), "error")
		httperror.InternalServerError(w, err)
		return
	}

	reader, err := os.OpenFile(req.Output, os.O_RDONLY, 0o600)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer closeWithLog(reader, "vith.httpThumbnail", req.Output)

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, reader); err != nil {
		logger.Error("unable to copy file: %s", err)
	}

	a.increaseMetric("http", "thumbnail", req.ItemType.String(), "success")
}

func videoThumbnail(req model.Request) error {
	var ffmpegOpts []string
	var customOpts []string

	if _, duration, err := getVideoDetailsFromName(req.Input); err != nil {
		logger.Error("unable to get container duration: %s", err)

		ffmpegOpts = append(ffmpegOpts, "-ss", "1.000")
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

	ffmpegOpts = append(ffmpegOpts, "-i", req.Input, "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150,fps=10", "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture", "-f", "webp")
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

func (a App) pdf(req model.Request) (err error) {
	var stats os.FileInfo
	stats, err = os.Stat(req.Input)
	if err != nil {
		return fmt.Errorf("unable to stats input file: %s", err)
	}

	var writer io.WriteCloser
	writer, err = os.OpenFile(req.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open output file: %s", err)
	}

	defer func() {
		if err == nil {
			return
		}

		if removeErr := os.Remove(req.Output); removeErr != nil {
			err = fmt.Errorf("%s: %w", err, removeErr)
		}
	}()

	defer closeWithLog(writer, "vith.pdf", req.Output)

	var reader io.ReadCloser
	reader, err = os.OpenFile(req.Input, os.O_RDONLY, 0o600) // file will be closed by streamPdf
	if err != nil {
		return fmt.Errorf("unable to open input file: %s", err)
	}

	return a.streamPdf(reader, writer, stats.Size())
}

func closeWithLog(closer io.Closer, fn, item string) {
	if err := closer.Close(); err != nil {
		logger.WithField("fn", fn).WithField("item", item).Error("unable to close: %s", err)
	}
}

func (a App) fileThumbnail(inputName string, output io.Writer, source string, itemType model.ItemType) error {
	reader, err := os.OpenFile(inputName, os.O_RDONLY, 0o600)
	if err != nil {
		a.increaseMetric(source, "thumbnail", itemType.String(), "file_error")
		return fmt.Errorf("unable to open input file: %s", err)
	}

	// PDF file are closed by request sender
	if itemType != model.TypePDF {
		defer closeWithLog(reader, "fileThumbnail", inputName)
	}

	switch itemType {
	case model.TypePDF:
		var info os.FileInfo
		info, err = os.Stat(inputName)
		if err != nil {
			a.increaseMetric(source, "thumbnail", itemType.String(), "file_error")
			return fmt.Errorf("unable to stat input file: %s", err)
		}

		err = a.streamPdf(reader, output, info.Size())
	case model.TypeImage:
		err = streamThumbnail(reader, output)
	}

	if err != nil {
		a.increaseMetric(source, "thumbnail", itemType.String(), "error")
		return err
	}

	a.increaseMetric(source, "thumbnail", itemType.String(), "success")
	return nil
}

func streamThumbnail(input io.Reader, output io.Writer) error {
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=150:150", "-vcodec", "libwebp", "-lossless", "0", "-compression_level", "6", "-q:v", "80", "-an", "-preset", "picture", "-f", "webp", "pipe:1")

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = buffer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", buffer.String(), err)
	}

	return nil
}

func (a App) streamPdf(input io.ReadCloser, output io.Writer, contentLength int64) error {
	r, err := a.imaginaryReq.Build(context.Background(), input)
	if err != nil {
		defer closeWithLog(input, "streamPdf", "")
		return fmt.Errorf("unable to build request: %s", err)
	}

	r.ContentLength = contentLength

	resp, err := request.DoWithClient(slowClient, r)
	if err != nil {
		return fmt.Errorf("unable to request imaginary: %s", err)
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err = io.CopyBuffer(output, resp.Body, buffer.Bytes()); err != nil {
		return fmt.Errorf("unable to copy imaginary response: %s", err)
	}

	return nil
}
