package config

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	stopChan chan struct{}
	once     sync.Once
	path     string
}

func NewWatcher(path string) *Watcher {
	if path == "" {
		path = "configs/config.yml"
	}
	return &Watcher{stopChan: make(chan struct{}), path: path}
}

func (w *Watcher) Start() {
	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error starting watcher: %v\n", err)
		return
	}

	go func() {
		defer func() {
			if w.watcher != nil {
				_ = w.watcher.Close()
			}
		}()

		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("modified file:", event)
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("error:", err)
			case <-w.stopChan:
				fmt.Println("Watcher stopped")
				return
			}
		}
	}()

	absPath, err := filepath.Abs(w.path)
	if err != nil {
		absPath = w.path
	}

	if err = w.watcher.Add(absPath); err != nil {
		fmt.Printf("Error adding watch path %s: %v\n", absPath, err)
		return
	}
}

func (w *Watcher) Stop() {
	w.once.Do(func() {
		close(w.stopChan)
		if w.watcher != nil {
			_ = w.watcher.Close()
		}
	})
}
