package config

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	stopChan chan struct{}
	once     sync.Once
	path     string
	config   *Config
}

type BackendChange struct {
	Added   []string
	Removed []string
}

func NewWatcher(path string, config *Config) *Watcher {
	if path == "" {
		path = "configs/config.yml"
	}
	return &Watcher{stopChan: make(chan struct{}), path: path, config: config}
}

func (w *Watcher) Start(changeChan chan BackendChange) {
	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error starting watcher: %v\n", err)
		return
	}

	const debounce = 30 * time.Second
	var timer *time.Timer

	go func() {
		defer func() {
			if w.watcher != nil {
				_ = w.watcher.Close()
			}
		}()

		for {
			var timerC <-chan time.Time
			if timer != nil {
				timerC = timer.C
			}

			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					if timer == nil {
						timer = time.NewTimer(debounce)
					} else {
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						timer.Reset(debounce)
					}
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("error:", err)
			case <-timerC:
				c, _ := Load(w.path)
				added, removed := CheckIfBackendChanged(c, w.config)
				if len(added) > 0 || len(removed) > 0 {
					changeChan <- BackendChange{Added: added, Removed: removed}
					w.config.Backends = c.Backends
				}
				timer = nil
			case <-w.stopChan:
				fmt.Println("Watcher stopped")
				if timer != nil {
					_ = timer.Stop()
					timer = nil
				}
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

func CheckIfBackendChanged(c *Config, prevConfig *Config) (added []string, removed []string) {
	if prevConfig == nil {
		return nil, nil
	}

	prevMap := make(map[string]struct{})
	for _, b := range prevConfig.Backends {
		prevMap[b.Url] = struct{}{}
	}

	currMap := make(map[string]struct{})
	for _, b := range c.Backends {
		currMap[b.Url] = struct{}{}
	}

	// Find added
	for u := range currMap {
		if _, ok := prevMap[u]; !ok {
			added = append(added, u)
		}
	}

	// Find removed
	for u := range prevMap {
		if _, ok := currMap[u]; !ok {
			removed = append(removed, u)
		}
	}

	return added, removed
}
