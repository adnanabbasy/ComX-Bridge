package cluster

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Role represents the node role.
type Role int

const (
	RolePrimary Role = iota
	RoleSecondary
)

// State represents the runtime state.
type State int

const (
	StateActive State = iota
	StateStandby
)

// Config holds cluster code configuration.
type Config struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Role     string        `yaml:"role" json:"role"` // "primary" or "secondary"
	PeerIP   string        `yaml:"peer_ip" json:"peer_ip"`
	Port     int           `yaml:"port" json:"port"`
	Interval time.Duration `yaml:"interval" json:"interval"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// Manager manages cluster state and failover.
type Manager struct {
	mu sync.RWMutex

	config Config
	role   Role
	state  State

	conn *net.UDPConn
	peer *net.UDPAddr

	onPromote func()
	onDemote  func()

	lastHeartbeat time.Time
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new cluster manager.
func NewManager(cfg Config) (*Manager, error) {
	role := RolePrimary
	if cfg.Role == "secondary" {
		role = RoleSecondary
	}

	state := StateActive
	if role == RoleSecondary {
		state = StateStandby
	}

	// Default timeouts
	if cfg.Interval == 0 {
		cfg.Interval = 1 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	return &Manager{
		config: cfg,
		role:   role,
		state:  state,
	}, nil
}

// SetCallbacks sets state change callbacks.
func (m *Manager) SetCallbacks(onPromote, onDemote func()) {
	m.onPromote = onPromote
	m.onDemote = onDemote
}

// Start starts the cluster manager.
func (m *Manager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	m.conn = conn

	if m.config.PeerIP != "" {
		m.peer, _ = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", m.config.PeerIP, m.config.Port))
	}

	go m.loop()

	if m.role == RolePrimary {
		fmt.Println("Cluster: Started as Primary (Active)")
	} else {
		fmt.Println("Cluster: Started as Secondary (Standby)")
	}

	return nil
}

// Stop stops the cluster manager.
func (m *Manager) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}

// IsActive returns true if this node is active.
func (m *Manager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == StateActive
}

func (m *Manager) loop() {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Read buffer
	buf := make([]byte, 1024)

	// Receiver routine
	go func() {
		for {
			if m.ctx.Err() != nil {
				return
			}
			m.conn.SetReadDeadline(time.Now().Add(m.config.Interval * 2))
			_, _, err := m.conn.ReadFromUDP(buf)
			if err == nil {
				m.mu.Lock()
				m.lastHeartbeat = time.Now()
				// If we were active but we are secondary and see heartbeat, demote?
				// Simple logic: Secondary just listens.
				m.mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			if m.role == RolePrimary {
				// Send Heartbeat
				if m.peer != nil {
					m.conn.WriteToUDP([]byte("ping"), m.peer)
				}
			} else {
				// Secondary checks timeout
				if time.Since(m.lastHeartbeat) > m.config.Timeout {
					if m.state == StateStandby {
						// Promote
						m.state = StateActive
						fmt.Println("Cluster: Heartbeat lost! Promoting to Active")
						if m.onPromote != nil {
							go m.onPromote()
						}
					}
				} else {
					// Heartbeat received recently
					if m.state == StateActive {
						// Demote (Primary recovered?)
						// For this simple implementation, we stick to Active until manual reset or specialized logic
						// But let's support auto-demote if Primary comes back?
						// Danger of flapping. Let's keep it simple: Stay Active for now.
					}
				}
			}
			m.mu.Unlock()
		}
	}
}
