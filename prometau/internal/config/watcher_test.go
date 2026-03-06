package config

import (
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	w := NewWatcher()
	if w == nil {
		t.Fatal("NewWatcher() returned nil")
	}
	if len(w.callbacks) != 0 {
		t.Errorf("expected empty callbacks, got %d", len(w.callbacks))
	}
}

func TestWatcherCallback(t *testing.T) {
	w := NewWatcher()

	called := false
	w.AddCallback(func() {
		called = true
	})

	w.TriggerCallbacks()

	// Wait for goroutine to execute
	time.Sleep(10 * time.Millisecond)

	if !called {
		t.Error("callback was not called")
	}
}

func TestMultipleCallbacks(t *testing.T) {
	w := NewWatcher()

	callCount := 0
	for i := 0; i < 5; i++ {
		w.AddCallback(func() {
			callCount++
		})
	}

	w.TriggerCallbacks()

	// Wait for goroutines to execute
	time.Sleep(50 * time.Millisecond)

	if callCount != 5 {
		t.Errorf("expected 5 callbacks, got %d", callCount)
	}
}

func TestClearCallbacks(t *testing.T) {
	w := NewWatcher()

	callCount := 0
	w.AddCallback(func() {
		callCount++
	})

	w.Clear()
	w.TriggerCallbacks()

	// Wait for goroutines
	time.Sleep(10 * time.Millisecond)

	if callCount != 0 {
		t.Errorf("expected 0 callbacks after clear, got %d", callCount)
	}
}

func TestConcurrentCallbacks(t *testing.T) {
	w := NewWatcher()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		w.AddCallback(func() {
			done <- true
		})
	}

	w.TriggerCallbacks()

	// Wait for all goroutines
	timeout := time.After(1 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("timeout waiting for callbacks")
		}
	}
}
