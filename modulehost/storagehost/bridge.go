package storagehost

import (
	"context"
	"fmt"
	"sync"
)

// Bridge is the minimal runtime bridge surface the storage host needs for
// sqlite/libsql-backed Mastra storage and vector stores.
type Bridge interface {
	URL() string
	// CloseContext releases the bridge before ctx expires. Implementations
	// should leave the bridge usable when returning an error.
	CloseContext(context.Context) error
}

// BridgeBuilder starts a bridge for a local storage path.
type BridgeBuilder func(path string) (Bridge, error)

var storageBridgeBuilders = struct {
	sync.RWMutex
	m map[string]BridgeBuilder
}{m: map[string]BridgeBuilder{}}

// RegisterBridgeBuilder links an optional storage bridge backend.
func RegisterBridgeBuilder(kind string, builder BridgeBuilder) {
	storageBridgeBuilders.Lock()
	defer storageBridgeBuilders.Unlock()
	if builder == nil {
		delete(storageBridgeBuilders.m, kind)
		return
	}
	storageBridgeBuilders.m[kind] = builder
}

func newBridge(kind, path string) (Bridge, error) {
	storageBridgeBuilders.RLock()
	builder := storageBridgeBuilders.m[kind]
	storageBridgeBuilders.RUnlock()
	if builder == nil {
		return nil, fmt.Errorf("storage bridge %q requires importing a backend package such as github.com/brainlet/brainkit/storagebridges/sqlite or the aggregate github.com/brainlet/brainkit/storagebridges", kind)
	}
	return builder(path)
}
