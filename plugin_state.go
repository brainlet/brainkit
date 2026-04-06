package brainkit

import (
	"context"
	"fmt"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"

	"github.com/nats-io/nats.go"
)

// PluginStateStore persists plugin state outside the local runtime.
type PluginStateStore interface {
	Get(context.Context, string, string) (string, error)
	Set(context.Context, string, string, string) error
	Close() error
}

type memoryPluginStateStore struct {
	mu    syncx.Mutex
	state map[string]string
}

func newMemoryPluginStateStore() *memoryPluginStateStore {
	return &memoryPluginStateStore{state: make(map[string]string)}
}

func (s *memoryPluginStateStore) Get(_ context.Context, pluginID, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[pluginStateKey(pluginID, key)], nil
}

func (s *memoryPluginStateStore) Set(_ context.Context, pluginID, key, value string) error {
	s.mu.Lock()
	s.state[pluginStateKey(pluginID, key)] = value
	s.mu.Unlock()
	return nil
}

func (s *memoryPluginStateStore) Close() error { return nil }

type natsPluginStateStore struct {
	conn *nats.Conn
	kv   nats.KeyValue
}

func newNATSPluginStateStore(cfg NodeConfig) (*natsPluginStateStore, error) {
	url := cfg.Messaging.NATSURL
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}
	name := cfg.Messaging.NATSName
	if name == "" {
		name = "brainkit"
	}
	conn, err := nats.Connect(url, nats.Name(name+"-plugin-state"))
	if err != nil {
		return nil, fmt.Errorf("brainkit: connect nats for plugin state: %w", err)
	}
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("brainkit: jetstream for plugin state: %w", err)
	}

	bucketName := sanitizeBucket("brainkit_" + cfg.Kernel.Namespace + "_plugin_state")
	kv, err := js.KeyValue(bucketName)
	if err != nil {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucketName})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("brainkit: create plugin state bucket: %w", err)
		}
	}

	return &natsPluginStateStore{conn: conn, kv: kv}, nil
}

func (s *natsPluginStateStore) Get(_ context.Context, pluginID, key string) (string, error) {
	entry, err := s.kv.Get(pluginStateKey(pluginID, key))
	if err == nats.ErrKeyNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(entry.Value()), nil
}

func (s *natsPluginStateStore) Set(_ context.Context, pluginID, key, value string) error {
	_, err := s.kv.Put(pluginStateKey(pluginID, key), []byte(value))
	return err
}

func (s *natsPluginStateStore) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}

func newPluginStateStore(cfg NodeConfig) (PluginStateStore, error) {
	// If KitStore is available, use it for persistent plugin state (survives restarts).
	if cfg.Kernel.Store != nil {
		return &kitStorePluginState{store: cfg.Kernel.Store}, nil
	}
	switch cfg.Messaging.Transport {
	case "nats":
		return newNATSPluginStateStore(cfg)
	default:
		return newMemoryPluginStateStore(), nil
	}
}

// kitStorePluginState wraps KitStore as a PluginStateStore.
type kitStorePluginState struct {
	store KitStore
}

func (s *kitStorePluginState) Get(_ context.Context, pluginID, key string) (string, error) {
	return s.store.LoadPluginState(pluginID, key)
}

func (s *kitStorePluginState) Set(_ context.Context, pluginID, key, value string) error {
	return s.store.SavePluginState(pluginID, key, value)
}

func (s *kitStorePluginState) Close() error { return nil }

func pluginStateKey(pluginID, key string) string {
	if pluginID == "" {
		return sanitizeBucket(key)
	}
	return sanitizeBucket(pluginID + "_" + key)
}

func sanitizeBucket(input string) string {
	replacer := strings.NewReplacer("/", "_", "@", "_", ".", "_", "-", "_", " ", "_", ":", "_")
	return replacer.Replace(input)
}
