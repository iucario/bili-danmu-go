package config

import (
	"sync"
)

// OnChangeFunc applies runtime side effects when config values change.
type OnChangeFunc func(prev, next *Config) error

// Store persists config changes and applies supported runtime updates.
type Store struct {
	mu       sync.RWMutex
	path     string
	current  Config
	onChange OnChangeFunc
}

// New returns a Store initialised with the current active config.
func New(path string, current *Config, onChange OnChangeFunc) *Store {
	initial := Default()
	if current != nil {
		initial = current
	}

	return &Store{
		path:     path,
		current:  *initial,
		onChange: onChange,
	}
}

// LoadEditable reads the config file values without environment overrides.
func (s *Store) LoadEditable() (*Config, error) {
	return LoadEditable(s.path)
}

// Get returns the currently active config after environment overrides.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Save writes the config file and applies any hot-reloadable runtime changes.
func (s *Store) Save(next Config) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := next.Save(s.path); err != nil {
		return nil, err
	}

	editable, err := LoadEditable(s.path)
	if err != nil {
		return nil, err
	}
	active, err := Load(s.path)
	if err != nil {
		return nil, err
	}

	if s.onChange != nil {
		err := s.onChange(&s.current, active)
		if err != nil {
			return nil, err
		}
	}

	s.current = *active
	return editable, nil
}
