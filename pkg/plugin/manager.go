package plugin

import (
	"fmt"
	"sync"
)

// SimpleManager implements a basic plugin manager.
type SimpleManager struct {
	mu       sync.RWMutex
	registry *Registry
	plugins  map[string]Plugin
	loader   Loader
	started  bool
}

// NewManager creates a new plugin manager.
func NewManager(loader Loader, registry *Registry) *SimpleManager {
	if registry == nil {
		registry = NewRegistry()
	}
	return &SimpleManager{
		registry: registry,
		plugins:  make(map[string]Plugin),
		loader:   loader,
	}
}

// Load loads a plugin from path.
func (m *SimpleManager) Load(path string) (Plugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, err := m.loader.LoadFile(path)
	if err != nil {
		return nil, err
	}

	info := p.Info()
	if _, exists := m.plugins[info.Name]; exists {
		return nil, ErrPluginExists
	}

	// Register based on type
	switch info.Type {
	case TypeTransport:
		if tp, ok := p.(TransportPlugin); ok {
			if err := m.registry.RegisterTransport(info.Name, tp); err != nil {
				return nil, err
			}
		} else {
			return nil, ErrPluginTypeMismatch
		}
	case TypeProtocol:
		if pp, ok := p.(ProtocolPlugin); ok {
			if err := m.registry.RegisterProtocol(info.Name, pp); err != nil {
				return nil, err
			}
		} else {
			return nil, ErrPluginTypeMismatch
		}
	}

	m.plugins[info.Name] = p
	return p, nil
}

// Unload unloads a plugin.
func (m *SimpleManager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return ErrPluginNotFound
	}

	if err := p.Stop(); err != nil {
		return err
	}

	delete(m.plugins, name)
	return nil
}

// Get retrieves a plugin by name.
func (m *SimpleManager) Get(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.plugins[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return p, nil
}

// List returns all loaded plugins.
func (m *SimpleManager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]Info, 0, len(m.plugins))
	for _, p := range m.plugins {
		infos = append(infos, p.Info())
	}
	return infos
}

// ListByType returns plugins of a specific type.
func (m *SimpleManager) ListByType(pluginType Type) []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]Info, 0)
	for _, p := range m.plugins {
		info := p.Info()
		if info.Type == pluginType {
			infos = append(infos, info)
		}
	}
	return infos
}

// Start starts all plugins.
func (m *SimpleManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	for _, p := range m.plugins {
		if err := p.Start(); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", p.Info().Name, err)
		}
	}

	m.started = true
	return nil
}

// Stop stops all plugins.
func (m *SimpleManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	for _, p := range m.plugins {
		if err := p.Stop(); err != nil {
			// Log error but continue
		}
	}

	m.started = false
	return nil
}

// Health returns the health of all plugins.
func (m *SimpleManager) Health() map[string]HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]HealthStatus)
	for name, p := range m.plugins {
		status[name] = p.Health()
	}
	return status
}
