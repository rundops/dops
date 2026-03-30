package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatcherCallback is called when catalog files change.
type WatcherCallback func()

// Watcher watches catalog directories for YAML changes and triggers
// a debounced reload callback.
type Watcher struct {
	watcher  *fsnotify.Watcher
	callback WatcherCallback
	debounce time.Duration
	done     chan struct{}
	mu       sync.Mutex
	timer    *time.Timer
}

// NewWatcher creates a file watcher on the given directories.
func NewWatcher(dirs []string, debounce time.Duration, callback WatcherCallback) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}

	w := &Watcher{
		watcher:  fw,
		callback: callback,
		debounce: debounce,
		done:     make(chan struct{}),
	}

	for _, dir := range dirs {
		if err := w.addRecursive(dir); err != nil {
			_ = fw.Close() // best-effort cleanup on init failure
			return nil, err
		}
	}

	go w.loop()
	return w, nil
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if isYAMLFile(event.Name) && isWriteEvent(event) {
				w.scheduleReload()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			_ = err // watcher errors are non-fatal; logging would corrupt MCP stdio
		case <-w.done:
			return
		}
	}
}

func (w *Watcher) scheduleReload() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		if w.callback != nil {
			w.callback()
		}
	})
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func isWriteEvent(e fsnotify.Event) bool {
	return e.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0
}
