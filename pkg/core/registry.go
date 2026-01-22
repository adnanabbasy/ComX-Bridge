package core

import (
	"fmt"
	"sort"
	"sync"

	"github.com/commatea/ComX-Bridge/pkg/protocol"
	"github.com/commatea/ComX-Bridge/pkg/transport"
)

// TransportRegistry implements transport.Registry.
type TransportRegistry struct {
	mu        sync.RWMutex
	factories map[string]transport.Factory
}

// NewTransportRegistry creates a new transport registry.
func NewTransportRegistry() *TransportRegistry {
	return &TransportRegistry{
		factories: make(map[string]transport.Factory),
	}
}

func (r *TransportRegistry) Register(factory transport.Factory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if factory == nil {
		return fmt.Errorf("factory is nil")
	}

	r.factories[factory.Type()] = factory
	return nil
}

func (r *TransportRegistry) Get(transportType string) (transport.Factory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.factories[transportType]
	if !ok {
		return nil, fmt.Errorf("transport factory not found: %s", transportType)
	}
	return f, nil
}

func (r *TransportRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func (r *TransportRegistry) Create(config transport.Config) (transport.Transport, error) {
	f, err := r.Get(config.Type)
	if err != nil {
		return nil, err
	}

	if err := f.Validate(config); err != nil {
		return nil, err
	}

	return f.Create(config)
}

// ProtocolRegistry implements protocol.Registry.
type ProtocolRegistry struct {
	mu        sync.RWMutex
	factories map[string]protocol.Factory
}

// NewProtocolRegistry creates a new protocol registry.
func NewProtocolRegistry() *ProtocolRegistry {
	return &ProtocolRegistry{
		factories: make(map[string]protocol.Factory),
	}
}

func (r *ProtocolRegistry) Register(factory protocol.Factory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if factory == nil {
		return fmt.Errorf("factory is nil")
	}

	r.factories[factory.Type()] = factory
	return nil
}

func (r *ProtocolRegistry) Get(protocolType string) (protocol.Factory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.factories[protocolType]
	if !ok {
		return nil, fmt.Errorf("protocol factory not found: %s", protocolType)
	}
	return f, nil
}

func (r *ProtocolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func (r *ProtocolRegistry) Create(config protocol.Config) (protocol.Protocol, error) {
	f, err := r.Get(config.Type)
	if err != nil {
		return nil, err
	}

	if err := f.Validate(config); err != nil {
		return nil, err
	}

	return f.Create(config)
}
