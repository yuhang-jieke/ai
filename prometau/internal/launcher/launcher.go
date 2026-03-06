// Package launcher provides multi-node HTTP server management.
package launcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// NodeStatus represents the status of a node.
type NodeStatus string

const (
	StatusStarting   NodeStatus = "starting"
	StatusRunning    NodeStatus = "running"
	StatusStopped    NodeStatus = "stopped"
	StatusFailed     NodeStatus = "failed"
	StatusRestarting NodeStatus = "restarting"
)

// Node represents a single HTTP server node.
type Node struct {
	Addr         string
	Server       *http.Server
	Status       NodeStatus
	RestartCount int
	lastError    error
	mu           sync.RWMutex
}

// GetStatus returns the current status of the node.
func (n *Node) GetStatus() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Status
}

// SetStatus sets the status of the node.
func (n *Node) SetStatus(status NodeStatus) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Status = status
}

// Launcher manages multiple HTTP server nodes.
type Launcher struct {
	config   *LauncherConfig
	nodes    map[string]*Node
	mu       sync.RWMutex
	wg       sync.WaitGroup
	stopChan chan struct{}
	stopOnce sync.Once
	handler  http.Handler
}

// LauncherConfig holds configuration for the launcher.
type LauncherConfig struct {
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxRestarts      int
	RestartBaseDelay time.Duration
	ShutdownTimeout  time.Duration
}

// DefaultLauncherConfig returns default launcher configuration.
func DefaultLauncherConfig() *LauncherConfig {
	return &LauncherConfig{
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		MaxRestarts:      3,
		RestartBaseDelay: time.Second,
		ShutdownTimeout:  30 * time.Second,
	}
}

// NewLauncher creates a new Launcher.
func NewLauncher(handler http.Handler, config *LauncherConfig) *Launcher {
	if config == nil {
		config = DefaultLauncherConfig()
	}

	return &Launcher{
		config:   config,
		nodes:    make(map[string]*Node),
		stopChan: make(chan struct{}),
		handler:  handler,
	}
}

// Launch starts HTTP server nodes.
// Single node: Launch("127.0.0.1:8080")
// Multi node: Launch("127.0.0.1:8081", "127.0.0.1:8082")
func (l *Launcher) Launch(addrs ...string) error {
	if len(addrs) == 0 {
		return fmt.Errorf("at least one address is required")
	}

	for _, addr := range addrs {
		if err := l.startNode(addr); err != nil {
			return fmt.Errorf("failed to start node %s: %w", addr, err)
		}
	}

	// Start monitoring
	go l.monitorNodes()

	slog.Info("launcher started", "nodes", len(addrs))
	return nil
}

// Shutdown gracefully stops all nodes.
func (l *Launcher) Shutdown() error {
	l.stopOnce.Do(func() {
		close(l.stopChan)
	})

	l.mu.RLock()
	defer l.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), l.config.ShutdownTimeout)
	defer cancel()

	for _, node := range l.nodes {
		if err := node.Server.Shutdown(ctx); err != nil {
			slog.Error("node shutdown error", "addr", node.Addr, "error", err)
		}
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("launcher stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	}
}

// GetStatus returns status of all nodes.
func (l *Launcher) GetStatus() []*NodeInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	infos := make([]*NodeInfo, 0, len(l.nodes))
	for addr, node := range l.nodes {
		infos = append(infos, &NodeInfo{
			Addr:         addr,
			Status:       node.GetStatus(),
			RestartCount: node.RestartCount,
			LastError:    node.lastError,
		})
	}

	return infos
}

// NodeInfo contains information about a node.
type NodeInfo struct {
	Addr         string     `json:"addr"`
	Status       NodeStatus `json:"status"`
	RestartCount int        `json:"restart_count"`
	LastError    error      `json:"last_error,omitempty"`
}

func (l *Launcher) startNode(addr string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.nodes[addr]; exists {
		return fmt.Errorf("node %s already exists", addr)
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      l.handler,
		ReadTimeout:  l.config.ReadTimeout,
		WriteTimeout: l.config.WriteTimeout,
	}

	node := &Node{
		Addr:   addr,
		Server: server,
		Status: StatusStarting,
	}

	l.nodes[addr] = node
	l.wg.Add(1)

	go l.runNode(node)

	slog.Info("node starting", "addr", addr)
	return nil
}

func (l *Launcher) runNode(node *Node) {
	defer l.wg.Done()

	for {
		node.SetStatus(StatusRunning)
		slog.Info("node listening", "addr", node.Addr)

		err := node.Server.ListenAndServe()

		// Check if this is a normal shutdown
		select {
		case <-l.stopChan:
			node.SetStatus(StatusStopped)
			slog.Info("node stopped", "addr", node.Addr)
			return
		default:
		}

		// Node failed
		node.SetStatus(StatusFailed)
		node.lastError = err
		slog.Error("node failed", "addr", node.Addr, "error", err)

		// Auto-restart with backoff
		if node.RestartCount < l.config.MaxRestarts {
			node.SetStatus(StatusRestarting)
			node.RestartCount++
			delay := l.getBackoffDelay(node.RestartCount)
			slog.Info("restarting node", "addr", node.Addr, "attempt", node.RestartCount, "delay", delay)
			time.Sleep(delay)
			continue
		}

		slog.Error("node stopped permanently", "addr", node.Addr, "restarts", node.RestartCount)
		return
	}
}

func (l *Launcher) monitorNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopChan:
			return
		case <-ticker.C:
			l.checkNodeHealth()
		}
	}
}

func (l *Launcher) checkNodeHealth() {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for addr, node := range l.nodes {
		if node.GetStatus() != StatusRunning {
			continue
		}

		// Simple health check via TCP connection
		resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
		if err != nil || resp.StatusCode != http.StatusOK {
			slog.Warn("node health check failed", "addr", addr)
			// Could trigger restart here
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

func (l *Launcher) getBackoffDelay(restartCount int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s...
	delay := l.config.RestartBaseDelay * time.Duration(1<<uint(restartCount-1))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}
