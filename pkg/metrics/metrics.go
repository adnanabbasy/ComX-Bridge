package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counters
	PacketCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "comx_gateway_packets_total",
		Help: "The total number of packets processed by gateways",
	}, []string{"gateway", "direction", "status"})

	ErrorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "comx_gateway_errors_total",
		Help: "The total number of errors in gateways",
	}, []string{"gateway", "type"})

	// Gauges
	ConnectedGateways = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "comx_connected_gateways_total",
		Help: "The total number of currently connected gateways",
	})
)

// Direction constants
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// Status constants
const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// IncPacket increments the packet counter.
func IncPacket(gateway, direction, status string) {
	PacketCount.WithLabelValues(gateway, direction, status).Inc()
}

// IncError increments the error counter.
func IncError(gateway, errType string) {
	ErrorCount.WithLabelValues(gateway, errType).Inc()
}

// SetConnectedGateways sets the number of connected gateways.
func SetConnectedGateways(count int) {
	ConnectedGateways.Set(float64(count))
}
