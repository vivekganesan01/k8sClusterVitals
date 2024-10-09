package helpers

import (
	"sync"
)

type SafeMap struct {
	m  map[string]string
	mu sync.RWMutex
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		m: make(map[string]string),
	}
}

// Set adds a key-value pair to the map
func (s *SafeMap) Set(key, value string) {
	s.mu.Lock() // Lock for writing
	defer s.mu.Unlock()
	s.m[key] = value
}

// Get retrieves the value for a given key
func (s *SafeMap) Get(key string) (string, bool) {
	s.mu.RLock() // Lock for reading
	defer s.mu.RUnlock()
	value, ok := s.m[key]
	return value, ok
}

// Delete removes a key from the map
func (s *SafeMap) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}
