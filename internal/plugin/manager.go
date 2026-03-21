package plugin

import (
	"log"
	"sync"
)

// Manager manages all plugin subprocesses for a Kit.
type Manager struct {
	bridge  Bridge
	plugins map[string]*conn
	mu      sync.Mutex
}

func NewManager(bridge Bridge) *Manager {
	return &Manager{
		bridge:  bridge,
		plugins: make(map[string]*conn),
	}
}

func (pm *Manager) StartAll(configs []Config) {
	for i := range configs {
		cfg := configs[i]
		ApplyDefaults(&cfg)
		if err := pm.startPlugin(cfg); err != nil {
			log.Printf("[plugin:%s] failed to start: %v", cfg.Name, err)
		}
	}
}

// GetConn returns the connection for a named plugin (for testing).
func (pm *Manager) GetConn(name string) *conn {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.plugins[name]
}
