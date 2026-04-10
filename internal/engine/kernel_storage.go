package engine

import (
	"fmt"

	"github.com/brainlet/brainkit/internal/libsql"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// AddStorage registers a new named storage at runtime.
// For sqlite: starts a libsql bridge + registers in provider registry.
// For others: registers in provider registry only.
func (k *Kernel) AddStorage(name string, cfg types.StorageConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if cfg.Type == "sqlite" {
		if _, exists := k.storages[name]; exists {
			return &sdk.AlreadyExistsError{Resource: "storage", Name: name}
		}
		srv, err := libsql.NewServer(cfg.Path)
		if err != nil {
			return err
		}
		k.storages[name] = srv
		reg := storageToRegistration(cfg, srv.URL())
		k.providers.RegisterStorage(name, reg)
	} else {
		reg := storageToRegistration(cfg, "")
		k.providers.RegisterStorage(name, reg)
	}
	return nil
}

// RemoveStorage stops and removes a named storage.
func (k *Kernel) RemoveStorage(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		_ = srv.Close()
		delete(k.storages, name)
	}
	k.providers.UnregisterStorage(name)
	return nil
}

// StorageURL returns the HTTP URL for a named storage bridge.
func (k *Kernel) StorageURL(name string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		return srv.URL()
	}
	return ""
}

// initStorages starts sqlite bridges for all sqlite storage entries.
// Must be called before loadRuntime — libsql servers need to be running.
// Returns a path→URL map for sqlite bridge sharing with vectors.
func (k *Kernel) initStorages(cfg types.KernelConfig) (map[string]string, error) {
	bridgeURLs := make(map[string]string)
	for name, scfg := range cfg.Storages {
		if scfg.Type == "sqlite" {
			srv, err := libsql.NewServer(scfg.Path)
			if err != nil {
				return nil, fmt.Errorf("storage %q: %w", name, err)
			}
			k.storages[name] = srv
			bridgeURLs[scfg.Path] = srv.URL()
		}
	}
	return bridgeURLs, nil
}

// registerStorages registers all storages in the provider registry.
func (k *Kernel) registerStorages(cfg types.KernelConfig, bridgeURLs map[string]string) {
	for name, scfg := range cfg.Storages {
		bridgeURL := ""
		if scfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[scfg.Path]
		}
		reg := storageToRegistration(scfg, bridgeURL)
		k.providers.RegisterStorage(name, reg)
	}
}

// registerVectors registers all vector stores in the provider registry.
// For sqlite vectors, reuses the bridge URL from a matching storage path.
func (k *Kernel) registerVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	for name, vcfg := range cfg.Vectors {
		bridgeURL := ""
		if vcfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[vcfg.Path]
			if bridgeURL == "" {
				srv, err := libsql.NewServer(vcfg.Path)
				if err != nil {
					return fmt.Errorf("vector %q: %w", name, err)
				}
				k.storages["vec_"+name] = srv
				bridgeURL = srv.URL()
				bridgeURLs[vcfg.Path] = bridgeURL
			}
		}
		reg := vectorToRegistration(vcfg, bridgeURL)
		k.providers.RegisterVectorStore(name, reg)
	}
	return nil
}
