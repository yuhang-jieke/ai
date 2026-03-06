// Package gospacex provides a simple API for HTTP server management.
package gospacex

import (
	"net/http"
	"sync"

	"github.com/yuhang-jieke/ai/internal/launcher"
)

var (
	httpInstance *HTTPServer
	once         sync.Once
)

// HTTPServer provides HTTP server management functionality.
type HTTPServer struct {
	launcher *launcher.Launcher
	handler  http.Handler
}

// httpServer is the global HTTP server instance.
var httpServer *HTTPServer

func init() {
	once.Do(func() {
		httpServer = &HTTPServer{}
	})
}

// Launch starts HTTP server nodes.
//
// Examples:
//
//	// Single node
//	gospacex.httpServer.Launch("127.0.0.1:8080")
//
//	// Multi-node
//	gospacex.httpServer.Launch("127.0.0.1:8081", "127.0.0.1:8082")
func (h *HTTPServer) Launch(addrs ...string) error {
	if h.launcher == nil {
		return ErrNotInitialized
	}
	return h.launcher.Launch(addrs...)
}

// Shutdown gracefully stops all nodes.
func (h *HTTPServer) Shutdown() error {
	if h.launcher == nil {
		return nil
	}
	return h.launcher.Shutdown()
}

// Status returns status of all nodes.
func (h *HTTPServer) Status() []*launcher.NodeInfo {
	if h.launcher == nil {
		return nil
	}
	return h.launcher.GetStatus()
}

// Configure initializes the HTTP server instance with custom configuration.
func (h *HTTPServer) Configure(handler http.Handler, config *launcher.LauncherConfig) {
	h.handler = handler
	h.launcher = launcher.NewLauncher(handler, config)
}

// SetHandler sets the HTTP handler.
func (h *HTTPServer) SetHandler(handler http.Handler) {
	h.handler = handler
}

// GetLauncher returns the underlying launcher.
func (h *HTTPServer) GetLauncher() *launcher.Launcher {
	return h.launcher
}

// ErrNotInitialized is returned when http is not initialized.
var ErrNotInitialized = &httpError{"HTTP not initialized. Call Configure() first."}

type httpError struct {
	msg string
}

func (e *httpError) Error() string {
	return e.msg
}

// HTTP returns the global HTTP server instance.
func HTTP() *HTTPServer {
	return httpServer
}
