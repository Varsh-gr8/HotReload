package watcher

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var ignoredDirs = map[string]bool{
	".git":        true,
	"node_modules": true,
	"bin":         true,
	"tmp":         true,
	"vendor":      true,
}

var watchedExtensions = map[string]bool{
	".go": true,
}

type Watcher struct {
	fsw      *fsnotify.Watcher
	logger   *slog.Logger
	debounce time.Duration
}

func New(logger *slog.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsw:      fsw,
		logger:   logger,
		debounce: 150 * time.Millisecond,
	}, nil
}

func (w *Watcher) AddRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			w.logger.Warn("skipping path during walk", "path", path, "error", err)
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if ignoredDirs[d.Name()] {
			w.logger.Info("ignoring directory", "dir", path)
			return filepath.SkipDir
		}

		w.logger.Info("watching directory", "dir", path)
		return w.fsw.Add(path)
	})
}

func (w *Watcher) Watch() (<-chan struct{}, error) {
	trigger := make(chan struct{}, 1)

	go func() {
		var timer *time.Timer

		for {
			select {
			case event, ok := <-w.fsw.Events:
				if !ok {
					return
				}

				if filepath.Ext(event.Name) != "" &&
					!watchedExtensions[filepath.Ext(event.Name)] {
					continue
				}

				if event.Op&fsnotify.Create != 0 {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						w.logger.Info("new directory detected, adding watch",
							"dir", event.Name,
						)
						w.AddRecursive(event.Name)
						continue
					}
				}

				if event.Op&fsnotify.Remove != 0 {
					w.logger.Info("path removed, cleaning watch",
						"path", event.Name,
					)
					w.fsw.Remove(event.Name)
					continue
				}

				w.logger.Info("file change detected",
					"file", event.Name,
					"op", event.Op.String(),
				)

				if timer != nil {
					timer.Stop()
				}

				timer = time.AfterFunc(w.debounce, func() {
					select {
					case trigger <- struct{}{}:
					default:
					}
				})

			case err, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
				w.logger.Error("watcher error", "error", err)
			}
		}
	}()

	return trigger, nil
}

func (w *Watcher) Close() {
	w.fsw.Close()
}
