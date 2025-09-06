package provider

import (
	"fmt"
	"sync"
)

// ProviderFactory creates a provider instance
type ProviderFactory func(config map[string]string) (Provider, error)

// Registry manages provider registration
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ProviderFactory
}

var globalRegistry = &Registry{
	providers: make(map[string]ProviderFactory),
}

// Register a new provider
func Register(name string, factory ProviderFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.providers[name] = factory
}

// Get a provider by name
func Get(name string, config map[string]string) (Provider, error) {
	globalRegistry.mu.RLock()
	factory, exists := globalRegistry.providers[name]
	globalRegistry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not registered", name)
	}

	return factory(config)
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalRegistry.providers))
	for name := range globalRegistry.providers {
		names = append(names, name)
	}
	return names
}