package plugin

import (
	"sync"
)

// Registry provides type-safe plugin lookup and management.
// It maintains separate registries for different plugin types.
type Registry struct {
	mu         sync.RWMutex
	transports map[string]TransportPlugin
	protocols  map[string]ProtocolPlugin
	ai         map[string]Plugin
	all        map[string]Plugin // All plugins by name
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		transports: make(map[string]TransportPlugin),
		protocols:  make(map[string]ProtocolPlugin),
		ai:         make(map[string]Plugin),
		all:        make(map[string]Plugin),
	}
}

// Register registers a plugin in the appropriate registry based on type.
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := plugin.Info()
	if _, exists := r.all[info.Name]; exists {
		return ErrPluginExists
	}

	// Register in type-specific registry
	switch info.Type {
	case TypeTransport:
		if tp, ok := plugin.(TransportPlugin); ok {
			r.transports[info.Name] = tp
		}
	case TypeProtocol:
		if pp, ok := plugin.(ProtocolPlugin); ok {
			r.protocols[info.Name] = pp
		}
	case TypeAI:
		r.ai[info.Name] = plugin
	}

	// Register in general registry
	r.all[info.Name] = plugin
	return nil
}

// Unregister removes a plugin from all registries.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, ok := r.all[name]
	if !ok {
		return ErrPluginNotFound
	}

	info := plugin.Info()
	switch info.Type {
	case TypeTransport:
		delete(r.transports, name)
	case TypeProtocol:
		delete(r.protocols, name)
	case TypeAI:
		delete(r.ai, name)
	}

	delete(r.all, name)
	return nil
}

// Get retrieves a plugin by name.
func (r *Registry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.all[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return plugin, nil
}

// RegisterTransport registers a transport plugin.
func (r *Registry) RegisterTransport(name string, plugin TransportPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transports[name]; exists {
		return ErrPluginExists
	}
	r.transports[name] = plugin
	r.all[name] = plugin
	return nil
}

// GetTransport retrieves a transport plugin.
func (r *Registry) GetTransport(name string) (TransportPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.transports[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return plugin, nil
}

// RegisterProtocol registers a protocol plugin.
func (r *Registry) RegisterProtocol(name string, plugin ProtocolPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.protocols[name]; exists {
		return ErrPluginExists
	}
	r.protocols[name] = plugin
	r.all[name] = plugin
	return nil
}

// GetProtocol retrieves a protocol plugin.
func (r *Registry) GetProtocol(name string) (ProtocolPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.protocols[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return plugin, nil
}

// ListTransports returns all registered transport plugin names.
func (r *Registry) ListTransports() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.transports))
	for name := range r.transports {
		names = append(names, name)
	}
	return names
}

// ListProtocols returns all registered protocol plugin names.
func (r *Registry) ListProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.protocols))
	for name := range r.protocols {
		names = append(names, name)
	}
	return names
}

// ListAll returns all registered plugin names.
func (r *Registry) ListAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.all))
	for name := range r.all {
		names = append(names, name)
	}
	return names
}

// ListByType returns plugins of a specific type.
func (r *Registry) ListByType(pluginType Type) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, plugin := range r.all {
		if plugin.Info().Type == pluginType {
			names = append(names, name)
		}
	}
	return names
}

// Count returns the total number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.all)
}

// Clear removes all plugins from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.transports = make(map[string]TransportPlugin)
	r.protocols = make(map[string]ProtocolPlugin)
	r.ai = make(map[string]Plugin)
	r.all = make(map[string]Plugin)
}
