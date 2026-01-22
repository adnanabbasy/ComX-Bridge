// Package ai - Auto Optimizer
// Provides automatic tuning of communication parameters based on runtime metrics.
package ai

import (
	"context"
	"sync"
	"time"
)

// OptimizerConfig holds configuration for the auto optimizer.
type OptimizerConfig struct {
	// Enabled enables auto optimization.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MinSamples is the minimum samples before making adjustments.
	MinSamples int `yaml:"min_samples" json:"min_samples"`

	// AdjustmentInterval is how often to check and adjust.
	AdjustmentInterval time.Duration `yaml:"adjustment_interval" json:"adjustment_interval"`

	// MaxTimeoutMs is the maximum timeout in milliseconds.
	MaxTimeoutMs int `yaml:"max_timeout_ms" json:"max_timeout_ms"`

	// MinTimeoutMs is the minimum timeout in milliseconds.
	MinTimeoutMs int `yaml:"min_timeout_ms" json:"min_timeout_ms"`

	// MaxRetries is the maximum retry count.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
}

// DefaultOptimizerConfig returns default optimizer settings.
func DefaultOptimizerConfig() OptimizerConfig {
	return OptimizerConfig{
		Enabled:            false,
		MinSamples:         100,
		AdjustmentInterval: 30 * time.Second,
		MaxTimeoutMs:       10000,
		MinTimeoutMs:       100,
		MaxRetries:         5,
	}
}

// AutoOptimizer automatically tunes communication parameters.
type AutoOptimizer struct {
	mu     sync.RWMutex
	config OptimizerConfig

	// Metrics tracking
	metrics *CommunicationMetrics

	// Current optimized parameters
	currentTimeout time.Duration
	currentRetries int

	// State
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// CommunicationMetrics tracks communication performance.
type CommunicationMetrics struct {
	mu sync.RWMutex

	// Response times
	responseTimes []time.Duration
	avgResponse   time.Duration
	p95Response   time.Duration
	p99Response   time.Duration

	// Error tracking
	totalRequests int64
	successCount  int64
	timeoutCount  int64
	errorCount    int64
	retryCount    int64

	// Throughput
	bytesReceived int64
	bytesSent     int64

	// Connection stats
	connectTime    time.Duration
	reconnectCount int64
}

// NewAutoOptimizer creates a new auto optimizer.
func NewAutoOptimizer(config OptimizerConfig) *AutoOptimizer {
	return &AutoOptimizer{
		config:         config,
		metrics:        &CommunicationMetrics{},
		currentTimeout: time.Duration(config.MaxTimeoutMs/2) * time.Millisecond,
		currentRetries: 3,
	}
}

// Start begins the optimization loop.
func (o *AutoOptimizer) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return nil
	}

	o.ctx, o.cancel = context.WithCancel(ctx)
	o.running = true

	go o.optimizationLoop()

	return nil
}

// Stop stops the optimizer.
func (o *AutoOptimizer) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return nil
	}

	if o.cancel != nil {
		o.cancel()
	}
	o.running = false

	return nil
}

// RecordRequest records a request metric.
func (o *AutoOptimizer) RecordRequest(duration time.Duration, success bool, wasRetry bool, wasTimeout bool) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.totalRequests++
	o.metrics.responseTimes = append(o.metrics.responseTimes, duration)

	// Keep only last 1000 samples
	if len(o.metrics.responseTimes) > 1000 {
		o.metrics.responseTimes = o.metrics.responseTimes[1:]
	}

	if success {
		o.metrics.successCount++
	}
	if wasTimeout {
		o.metrics.timeoutCount++
	}
	if wasRetry {
		o.metrics.retryCount++
	}
	if !success && !wasTimeout {
		o.metrics.errorCount++
	}
}

// RecordBytes records bytes transferred.
func (o *AutoOptimizer) RecordBytes(sent, received int64) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.bytesSent += sent
	o.metrics.bytesReceived += received
}

// GetRecommendedTimeout returns the recommended timeout.
func (o *AutoOptimizer) GetRecommendedTimeout() time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentTimeout
}

// GetRecommendedRetries returns the recommended retry count.
func (o *AutoOptimizer) GetRecommendedRetries() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentRetries
}

// GetMetrics returns current metrics summary.
func (o *AutoOptimizer) GetMetrics() MetricsSummary {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()

	successRate := float64(0)
	if o.metrics.totalRequests > 0 {
		successRate = float64(o.metrics.successCount) / float64(o.metrics.totalRequests) * 100
	}

	return MetricsSummary{
		TotalRequests:  o.metrics.totalRequests,
		SuccessRate:    successRate,
		AvgResponseMs:  o.metrics.avgResponse.Milliseconds(),
		P95ResponseMs:  o.metrics.p95Response.Milliseconds(),
		TimeoutCount:   o.metrics.timeoutCount,
		RetryCount:     o.metrics.retryCount,
		BytesSent:      o.metrics.bytesSent,
		BytesReceived:  o.metrics.bytesReceived,
		ReconnectCount: o.metrics.reconnectCount,
	}
}

// MetricsSummary is a summary of communication metrics.
type MetricsSummary struct {
	TotalRequests  int64   `json:"total_requests"`
	SuccessRate    float64 `json:"success_rate"`
	AvgResponseMs  int64   `json:"avg_response_ms"`
	P95ResponseMs  int64   `json:"p95_response_ms"`
	TimeoutCount   int64   `json:"timeout_count"`
	RetryCount     int64   `json:"retry_count"`
	BytesSent      int64   `json:"bytes_sent"`
	BytesReceived  int64   `json:"bytes_received"`
	ReconnectCount int64   `json:"reconnect_count"`
}

// optimizationLoop runs the periodic optimization.
func (o *AutoOptimizer) optimizationLoop() {
	ticker := time.NewTicker(o.config.AdjustmentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.optimize()
		}
	}
}

// optimize performs one optimization cycle.
func (o *AutoOptimizer) optimize() {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	samples := len(o.metrics.responseTimes)
	if samples < o.config.MinSamples {
		return // Not enough data
	}

	// Calculate statistics
	o.calculateStats()

	o.mu.Lock()
	defer o.mu.Unlock()

	// Timeout optimization
	// If too many timeouts, increase timeout
	timeoutRate := float64(o.metrics.timeoutCount) / float64(o.metrics.totalRequests)
	if timeoutRate > 0.05 { // More than 5% timeouts
		newTimeout := o.currentTimeout + (100 * time.Millisecond)
		if newTimeout.Milliseconds() <= int64(o.config.MaxTimeoutMs) {
			o.currentTimeout = newTimeout
		}
	} else if timeoutRate < 0.01 && o.metrics.avgResponse > 0 {
		// Very few timeouts, can reduce timeout based on p99
		// Set timeout to 2x p99 response time
		newTimeout := o.metrics.p99Response * 2
		if newTimeout.Milliseconds() >= int64(o.config.MinTimeoutMs) {
			o.currentTimeout = newTimeout
		}
	}

	// Retry optimization
	// If success rate is high after retries, reduce retries
	// If error rate is high, consider increasing retries
	successRate := float64(o.metrics.successCount) / float64(o.metrics.totalRequests)
	if successRate > 0.99 && o.currentRetries > 1 {
		o.currentRetries--
	} else if successRate < 0.95 && o.currentRetries < o.config.MaxRetries {
		o.currentRetries++
	}
}

// calculateStats calculates response time percentiles.
func (o *AutoOptimizer) calculateStats() {
	if len(o.metrics.responseTimes) == 0 {
		return
	}

	// Simple average
	var total time.Duration
	for _, d := range o.metrics.responseTimes {
		total += d
	}
	o.metrics.avgResponse = total / time.Duration(len(o.metrics.responseTimes))

	// Sort for percentiles (simple approach)
	sorted := make([]time.Duration, len(o.metrics.responseTimes))
	copy(sorted, o.metrics.responseTimes)

	// Simple bubble sort for small samples
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// P95
	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	o.metrics.p95Response = sorted[p95Index]

	// P99
	p99Index := int(float64(len(sorted)) * 0.99)
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}
	o.metrics.p99Response = sorted[p99Index]
}
