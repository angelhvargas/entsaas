package bootstrap

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// Component defines the lifecycle of a pluggable system component.
type Component interface {
	Name() string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// LifecycleManager orchestrates the boot loop and graceful shutdown of components.
type LifecycleManager struct {
	mu         sync.Mutex
	components []Component
	started    []Component
}

// NewLifecycleManager creates a new empty LifecycleManager.
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		components: make([]Component, 0),
		started:    make([]Component, 0),
	}
}

// Register adds a component to the lifecycle manager.
func (m *LifecycleManager) Register(c Component) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.components = append(m.components, c)
	log.Debug().Str("component", c.Name()).Msg("Registered component")
}

// InitAll initializes all registered components in order.
func (m *LifecycleManager) InitAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.components {
		log.Info().Str("component", c.Name()).Msg("Initializing component...")
		if err := c.Init(ctx); err != nil {
			return fmt.Errorf("component %q failed to initialize: %w", c.Name(), err)
		}
		log.Info().Str("component", c.Name()).Msg("Component initialized")
	}
	return nil
}

// StartAll starts all registered components in order.
func (m *LifecycleManager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.components {
		log.Info().Str("component", c.Name()).Msg("Starting component...")
		if err := c.Start(ctx); err != nil {
			// Stop already started components on startup failure
			m.stopAllStarted(ctx)
			return fmt.Errorf("component %q failed to start: %w", c.Name(), err)
		}
		m.started = append(m.started, c)
		log.Info().Str("component", c.Name()).Msg("Component started")
	}
	return nil
}

// StopAll gracefully shuts down all started components in REVERSE order.
func (m *LifecycleManager) StopAll(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopAllStarted(ctx)
}

func (m *LifecycleManager) stopAllStarted(ctx context.Context) {
	for i := len(m.started) - 1; i >= 0; i-- {
		c := m.started[i]
		log.Info().Str("component", c.Name()).Msg("Stopping component...")
		if err := c.Stop(ctx); err != nil {
			log.Error().Err(err).Str("component", c.Name()).Msg("Component failed to stop gracefully")
		} else {
			log.Info().Str("component", c.Name()).Msg("Component stopped")
		}
	}
	m.started = nil
}
