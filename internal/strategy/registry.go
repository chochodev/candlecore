package strategy

import (
	"fmt"
	"sync"
)

// StrategyFactory is a function that creates a new strategy instance
type StrategyFactory func() IStrategy

// Registry manages available trading strategies
type Registry struct {
	strategies map[string]StrategyFactory
	mu         sync.RWMutex
}

var globalRegistry = &Registry{
	strategies: make(map[string]StrategyFactory),
}

// Register adds a strategy to the global registry
func Register(name string, factory StrategyFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.strategies[name] = factory
}

// Get retrieves a strategy by name
func Get(name string) (IStrategy, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	factory, exists := globalRegistry.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy '%s' not found", name)
	}
	
	return factory(), nil
}

// List returns all registered strategy names
func List() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	names := make([]string, 0, len(globalRegistry.strategies))
	for name := range globalRegistry.strategies {
		names = append(names, name)
	}
	return names
}

// GetInfo returns information about a strategy without creating an instance
func GetInfo(name string) (string, string, error) {
	strategy, err := Get(name)
	if err != nil {
		return "", "", err
	}
	return strategy.GetName(), strategy.GetVersion(), nil
}
