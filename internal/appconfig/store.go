package appconfig

import (
	"sync"
)

// AppConfig holds user-configurable runtime settings.
// Add fields here as new admin-configurable items are introduced (e.g. chat filters).
type AppConfig struct{}

// Store is a thread-safe in-memory config store.
type Store struct {
	mu  sync.RWMutex
	cfg AppConfig
}

// New returns a Store initialised with default values.
func New() *Store {
	return &Store{}
}

// Get returns a snapshot of the current config.
func (s *Store) Get() AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Set replaces the entire config after validating all fields.
func (s *Store) Set(cfg AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

// Patch merges non-zero fields from patch into the current config.
func (s *Store) Patch(patch AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = patch
	return nil
}
