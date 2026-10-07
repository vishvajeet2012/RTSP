package websocket

import "sync"

type Manager struct {
	mu         sync.RWMutex
	hubs       map[string]*Hub
	maxClients int
}

func NewManager(maxClients int) *Manager {
	return &Manager{hubs: make(map[string]*Hub), maxClients: maxClients}
}
func (m *Manager) Create(id string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := NewHub(m.maxClients)
	m.hubs[id] = h
	return h
}
func (m *Manager) Get(id string) (*Hub, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hubs[id]
	return h, ok
}
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	h := m.hubs[id]
	delete(m.hubs, id)
	m.mu.Unlock()
	if h != nil {
		h.Close()
	}
}
