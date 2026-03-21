package brainkit

import (
	"fmt"

	"github.com/brainlet/brainkit/libsql"
)

// AddStorage starts a new named embedded SQLite storage and makes it available to JS.
// JS code can then use `new LibSQLStore({ id: "x", storage: "name" })`.
func (k *Kit) AddStorage(name string, cfg StorageConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, exists := k.storages[name]; exists {
		return fmt.Errorf("brainkit: storage %q already exists", name)
	}
	return k.addStorageInternal(name, cfg)
}

// RemoveStorage stops and removes a named storage.
func (k *Kit) RemoveStorage(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	srv, ok := k.storages[name]
	if !ok {
		return fmt.Errorf("brainkit: storage %q not found", name)
	}
	srv.Close()
	delete(k.storages, name)
	// Update JS-side storage map
	k.bridge.Eval("__storage_remove.js", fmt.Sprintf(
		`delete globalThis.__brainkit_storages[%q]`, name,
	))
	return nil
}

// StorageURL returns the HTTP URL for a named storage bridge.
// Returns "" if the storage doesn't exist.
func (k *Kit) StorageURL(name string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		return srv.URL()
	}
	return ""
}

func (k *Kit) addStorageInternal(name string, cfg StorageConfig) error {
	srv, err := libsql.NewServer(cfg.Path)
	if err != nil {
		return err
	}
	k.storages[name] = srv
	// Register in JS-side storage map
	k.bridge.Eval("__storage_add.js", fmt.Sprintf(
		`if (!globalThis.__brainkit_storages) globalThis.__brainkit_storages = {};
		 globalThis.__brainkit_storages[%q] = %q;`, name, srv.URL(),
	))
	return nil
}
