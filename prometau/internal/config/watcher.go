package config

import (
	"sync"
)

// Watcher manages configuration change callbacks.
type Watcher struct {
	mu        sync.RWMutex
	callbacks []func()
}

// NewWatcher creates a new configuration watcher.
func NewWatcher() *Watcher {
	return &Watcher{
		callbacks: make([]func(), 0),
	}
}

// AddCallback registers a callback to be called when configuration changes.
func (w *Watcher) AddCallback(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// TriggerCallbacks calls all registered callbacks.
// Each callback is called in a separate goroutine to avoid blocking.
func (w *Watcher) TriggerCallbacks() {
	w.mu.RLock()
	callbacks := make([]func(), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, cb := range callbacks {
		go cb()
	}
}

// Clear removes all registered callbacks.
func (w *Watcher) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = make([]func(), 0)
}
