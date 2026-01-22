package ai

import (
	"context"
	"math"
	"sync"
	"time"
)

// StatisticalDetector implements AnomalyDetector using statistical analysis.
// It tracks packet sizes and inter-arrival times to establish a baseline.
type StatisticalDetector struct {
	mu sync.RWMutex

	// Statistics
	packetCount uint64
	sizeSum     uint64
	sizeSqSum   uint64 // Sum of squares for variance

	// Thresholds (Z-score)
	threshold float64

	// State
	lastPacketTime time.Time
}

// NewStatisticalDetector creates a new statistical anomaly detector.
func NewStatisticalDetector() *StatisticalDetector {
	return &StatisticalDetector{
		threshold: 3.0, // Default 3-sigma
	}
}

// DetectAnomaly analyzes a stream of packets for anomalies.
func (d *StatisticalDetector) DetectAnomaly(ctx context.Context, stream <-chan []byte) (<-chan Anomaly, error) {
	out := make(chan Anomaly)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-stream:
				if !ok {
					return
				}

				result, _ := d.AnalyzePacket(ctx, data)
				if result != nil && result.IsAnomaly && len(result.Anomalies) > 0 {
					// Stream the first anomaly found
					out <- result.Anomalies[0]
				}
			}
		}
	}()

	return out, nil
}

// AnalyzePacket analyzes a single packet.
func (d *StatisticalDetector) AnalyzePacket(ctx context.Context, data []byte) (*AnomalyResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	size := float64(len(data))
	now := time.Now()

	// Update stats
	d.packetCount++
	d.sizeSum += uint64(size)
	d.sizeSqSum += uint64(size * size)

	// Calculate Mean and StdDev
	mean := float64(d.sizeSum) / float64(d.packetCount)
	variance := (float64(d.sizeSqSum) / float64(d.packetCount)) - (mean * mean)
	stdDev := math.Sqrt(variance)

	// Anomaly Check (only if we have enough samples, e.g., > 10)
	isAnomaly := false
	reason := ""
	severity := 0.0

	if d.packetCount > 10 && stdDev > 0 {
		zScore := math.Abs(size-mean) / stdDev
		if zScore > d.threshold {
			isAnomaly = true
			reason = "Packet size deviation"
			severity = zScore / d.threshold // Normalized severity
			if severity > 1.0 {
				severity = 1.0
			}
		}
	}

	// Inter-arrival check
	if !d.lastPacketTime.IsZero() {
		// Simple logic: if too fast or too slow?
		// For now, implementing simple size-based check.
	}
	d.lastPacketTime = now

	var anomalies []Anomaly
	if isAnomaly {
		anomalies = append(anomalies, Anomaly{
			Type:        AnomalyValue, // Use AnomalyValue for size deviation
			Severity:    SeverityWarning,
			Description: reason,
			Score:       severity,
			Timestamp:   now,
			Data:        data,
		})
	}

	return &AnomalyResult{
		IsAnomaly: isAnomaly,
		Score:     severity,
		Anomalies: anomalies,
	}, nil
}

// LearnNormalPattern learns from a set of samples.
func (d *StatisticalDetector) LearnNormalPattern(ctx context.Context, samples [][]byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Reset stats? Or accumulative? Let's reset for fresh learning.
	d.packetCount = 0
	d.sizeSum = 0
	d.sizeSqSum = 0

	for _, sample := range samples {
		size := float64(len(sample))
		d.packetCount++
		d.sizeSum += uint64(size)
		d.sizeSqSum += uint64(size * size)
	}

	return nil
}

// SetThreshold sets the sensitivity threshold.
func (d *StatisticalDetector) SetThreshold(threshold float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.threshold = threshold
}
