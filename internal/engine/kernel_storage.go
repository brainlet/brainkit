package engine

import (
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/storagehost"
)

func (k *Kernel) AddStorage(name string, cfg types.StorageConfig) error {
	return k.storageHost.AddStorage(name, cfg)
}

func (k *Kernel) RemoveStorage(name string) error {
	return k.storageHost.RemoveStorage(name)
}

func (k *Kernel) StorageURL(name string) string {
	return k.storageHost.URL(name)
}

func (k *Kernel) StorageManager() *storagehost.Manager {
	return k.storageHost
}

// initStorages starts sqlite bridges for all sqlite storage entries.
// Must be called before loadRuntime — libsql servers need to be running.
// Returns a path→URL map for sqlite bridge sharing with vectors.
func (k *Kernel) initStorages(cfg types.KernelConfig) (map[string]string, error) {
	return k.storageHost.InitBridges(cfg)
}

// registerStorages registers all storages in the provider registry.
func (k *Kernel) registerStorages(cfg types.KernelConfig, bridgeURLs map[string]string) {
	k.storageHost.RegisterStorages(cfg, bridgeURLs)
}

// registerVectors registers all vector stores in the provider registry.
// For sqlite vectors, reuses the bridge URL from a matching storage path.
func (k *Kernel) registerVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	return k.storageHost.RegisterVectors(cfg, bridgeURLs)
}
