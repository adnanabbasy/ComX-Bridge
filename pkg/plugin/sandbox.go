package plugin

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
)

var (
	// ErrSandboxTimeout is returned when sandbox execution exceeds timeout.
	ErrSandboxTimeout = errors.New("sandbox execution timeout")
	// ErrSandboxPanic is returned when sandbox execution panics.
	ErrSandboxPanic = errors.New("sandbox execution panic")
	// ErrSandboxMemoryLimit is returned when memory limit is exceeded.
	ErrSandboxMemoryLimit = errors.New("sandbox memory limit exceeded")
)

// SimpleSandbox provides basic plugin isolation with resource limits.
// This is a simplified implementation suitable for Go plugins.
// For stronger isolation, consider using OS-level containers.
type SimpleSandbox struct {
	mu           sync.Mutex
	timeout      time.Duration
	memoryLimit  int64
	cpuLimit     float64
	panicHandler func(interface{})
}

// NewSandbox creates a new simple sandbox with default limits.
func NewSandbox() *SimpleSandbox {
	return &SimpleSandbox{
		timeout:     30 * time.Second, // Default 30s timeout
		memoryLimit: 0,                // No limit by default
		cpuLimit:    0,                // No limit by default
	}
}

// Run executes a function within the sandbox with configured limits.
func (s *SimpleSandbox) Run(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Channel to receive result
	resultCh := make(chan error, 1)
	panicCh := make(chan interface{}, 1)

	// Track memory before execution
	var memBefore runtime.MemStats
	if s.memoryLimit > 0 {
		runtime.ReadMemStats(&memBefore)
	}

	// Run function in goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		resultCh <- fn()
	}()

	// Wait for completion or timeout
	select {
	case err := <-resultCh:
		// Check memory usage after execution
		if s.memoryLimit > 0 {
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)
			memUsed := int64(memAfter.Alloc - memBefore.Alloc)
			if memUsed > s.memoryLimit {
				return fmt.Errorf("%w: used %d bytes, limit %d bytes",
					ErrSandboxMemoryLimit, memUsed, s.memoryLimit)
			}
		}
		return err

	case panicVal := <-panicCh:
		if s.panicHandler != nil {
			s.panicHandler(panicVal)
		}
		return fmt.Errorf("%w: %v", ErrSandboxPanic, panicVal)

	case <-ctx.Done():
		return fmt.Errorf("%w: exceeded %v", ErrSandboxTimeout, s.timeout)
	}
}

// SetTimeout sets the maximum execution time for sandbox runs.
func (s *SimpleSandbox) SetTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeout = timeout
}

// SetMemoryLimit sets the maximum memory allocation allowed.
// Note: This is a soft limit checked after execution. For hard limits,
// use cgroups or container-based isolation.
func (s *SimpleSandbox) SetMemoryLimit(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryLimit = bytes
}

// SetCPULimit sets the maximum CPU usage percentage.
// Note: This is not enforced in the simple implementation.
// For real CPU limits, use OS-level resource controls.
func (s *SimpleSandbox) SetCPULimit(percent float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cpuLimit = percent
	// TODO: Implement CPU throttling if needed
}

// SetPanicHandler sets a custom panic handler.
func (s *SimpleSandbox) SetPanicHandler(handler func(interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicHandler = handler
}

// GetTimeout returns the current timeout setting.
func (s *SimpleSandbox) GetTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timeout
}

// GetMemoryLimit returns the current memory limit.
func (s *SimpleSandbox) GetMemoryLimit() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memoryLimit
}

// GetCPULimit returns the current CPU limit.
func (s *SimpleSandbox) GetCPULimit() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cpuLimit
}
