package vith

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/vith/pkg/model"
)

// Done close when work is over
func (a App) Done() <-chan struct{} {
	return a.done
}

// Start worker
func (a App) Start(done <-chan struct{}) {
	defer close(a.done)
	defer close(a.streamRequestQueue)
	defer a.stopOnce()

	if !a.storageApp.Enabled() {
		return
	}

	go func() {
		defer a.stopOnce()

		select {
		case <-done:
		case <-a.done:
		}
	}()

	for req := range a.streamRequestQueue {
		if err := a.generateStream(req); err != nil {
			logger.Error("unable to generate stream: %s", err)
		}
	}
}

func (a App) stopOnce() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
}

func (a App) generateStream(req model.Request) error {
	log := logger.WithField("input", req.Input).WithField("output", req.Output)
	log.Info("Generating stream...")

	inputName, finalizeInput, err := a.getInputVideoName(req.Input)
	if err != nil {
		return fmt.Errorf("unable to get input video name: %s", err)
	}
	defer finalizeInput()

	outputName, finalizeStream, err := a.getOutputStreamName(req.Output)
	if err != nil {
		return fmt.Errorf("unable to get video filename: %s", err)
	}
	defer finalizeStream()

	cmd := exec.Command("ffmpeg", "-i", inputName, "-codec:v", "libx264", "-preset", "superfast", "-codec:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "hls", "-hls_time", "4", "-hls_playlist_type", "event", "-hls_flags", "independent_segments", "-threads", "2", outputName)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	buffer.Reset()
	cmd.Stdout = buffer
	cmd.Stderr = buffer

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("unable to generate stream video: %s\n%s", err, buffer.Bytes())

		if cleanErr := a.cleanLocalStream(outputName); cleanErr != nil {
			err = fmt.Errorf("unable to remove generated files: %s: %w", cleanErr, err)
		}

		return err
	}

	log.Info("Generation succeeded!")
	return nil
}

func (a App) isValidStreamName(streamName string, shouldExist bool) error {
	if len(streamName) == 0 {
		return errors.New("name is required")
	}

	if filepath.Ext(streamName) != hlsExtension {
		return fmt.Errorf("only `%s` files are allowed", hlsExtension)
	}

	info, err := a.storageApp.Info(streamName)
	if shouldExist {
		if err != nil || info.IsDir {
			return fmt.Errorf("input `%s` doesn't exist or is a directory", streamName)
		}
	} else if err == nil {
		return fmt.Errorf("input `%s` already exists", streamName)
	}

	return nil
}

func (a App) getOutputStreamName(name string) (localName string, onEnd func(), err error) {
	onEnd = func() {}

	switch a.storageApp.Name() {
	case filesystem.Name:
		localName = a.storageApp.Path(name)
		return

	case s3.Name:
		localName = filepath.Join(a.tmpFolder, path.Base(name))

		onEnd = func() {
			manifest, err := a.storageApp.WriterTo(name)
			if err != nil {
				logger.Error("unable to get writer for manifest `%s`: %s", name, err)
				return
			}

			if err = copyLocalFile(localName, manifest); err != nil {
				logger.Error("unable to copy manifest to `%s`: %s", name, err)
				return
			}

			baseHlsName := strings.TrimSuffix(localName, hlsExtension)
			segments, err := filepath.Glob(fmt.Sprintf("%s*.ts", baseHlsName))
			if err != nil {
				logger.Error("unable to list hls segments for `%s`: %s", baseHlsName, err)
				return
			}

			outputDir := path.Dir(name)

			for _, file := range segments {
				segmentName := path.Join(outputDir, filepath.Base(file))
				segment, err := a.storageApp.WriterTo(segmentName)
				if err != nil {
					logger.Error("unable to get writer for segment `%s`: %s", segmentName, err)
					return
				}

				if err = copyLocalFile(file, segment); err != nil {
					logger.Error("unable to copy segment to `%s`: %s", segmentName, err)
					return
				}
			}

			if cleanErr := a.cleanLocalStream(localName); cleanErr != nil {
				logger.Error("unable to clean stream for `%s`: %s", localName, err)
				return
			}
		}
		return

	default:
		err = fmt.Errorf("unknown storage app")
		return
	}
}

func (a App) cleanLocalStream(name string) error {
	return a.cleanStream(name, os.Remove, filepath.Glob, "*.ts")
}
