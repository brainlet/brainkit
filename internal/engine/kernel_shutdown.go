package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Shutdown drains in-flight handlers, then closes everything.
// The context controls the drain timeout — when ctx expires, force-close proceeds.
func (k *Kernel) Shutdown(ctx context.Context) error {
	k.draining.Store(true)
	k.audit.HealthChanged("kit", "draining", true)
	k.waitForDrain(ctx)
	k.audit.HealthChanged("kit", "shutdown", false)
	return k.close()
}

// Close shuts down with a short drain timeout (5s).
func (k *Kernel) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return k.Shutdown(ctx)
}

// resolveSecretStore determines the secret store from config with clear precedence:
// 1. Explicit SecretStore → use it
// 2. types.SQLiteStore + SecretKey → encrypted KV store
// 3. types.SQLiteStore + no SecretKey → unencrypted KV store (dev mode, logged warning)
// 4. No types.SQLiteStore → environment variable fallback
func resolveSecretStore(cfg types.KernelConfig, logger *slog.Logger) secrets.SecretStore {
	if cfg.SecretStore != nil {
		return cfg.SecretStore
	}

	key := cfg.SecretKey
	if key == "" {
		key = os.Getenv("BRAINKIT_SECRET_KEY")
	}

	// Need a *types.SQLiteStore to back the encrypted KV store
	sqliteStore, hasSQLite := cfg.Store.(*types.SQLiteStore)
	if !hasSQLite || sqliteStore == nil {
		return secrets.NewEnvStore()
	}

	if key == "" {
		logger.Warn("SecretKey not set, secrets stored without encryption")
	}

	store, err := secrets.NewEncryptedKVStore(sqliteStore.DB, key)
	if err != nil {
		types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "CreateEncryptedSecretStore", Cause: err,
		}, types.ErrorContext{Operation: "CreateEncryptedSecretStore", Component: "kernel"})
		return secrets.NewEnvStore()
	}
	return store
}

// close is the internal shutdown logic.
func (k *Kernel) close() error {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return nil
	}
	k.closed = true
	subs := make([]func(), 0, len(k.bridgeSubs))
	for _, cancel := range k.bridgeSubs {
		subs = append(subs, cancel)
	}
	k.bridgeSubs = map[string]func(){}
	k.mu.Unlock()

	for _, cancel := range subs {
		cancel()
	}

	// Stop all schedule timers
	k.mu.Lock()
	for _, entry := range k.schedules {
		entry.timer.Stop()
	}
	k.schedules = nil
	k.mu.Unlock()

	// Stop all stream heartbeat goroutines
	if k.streamTracker != nil {
		k.streamTracker.CloseAll()
	}

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Shut down router first (stops processing messages)
	if k.router != nil {
		collect(k.router.Close())
	}

	if k.agentsDomain != nil && k.agents != nil {
		k.agentsDomain.UnregisterAllForKit(k.agents.ID())
	}
	// Close modules (reverse init order)
	for i := len(k.modules) - 1; i >= 0; i-- {
		collect(k.modules[i].Close())
	}
	if k.mcp != nil {
		collect(k.mcp.Close())
	}
	if k.config.Store != nil {
		collect(k.config.Store.Close())
	}
	if k.agents != nil {
		k.agents.Close()
	}
	for name, srv := range k.storages {
		if err := srv.Close(); err != nil {
			collect(fmt.Errorf("storage %q: %w", name, err))
		}
	}

	// Shut down transport last (only if we own it — Node owns its own)
	if k.ownsTransport && k.transport != nil {
		collect(k.transport.Close())
	}

	return firstErr
}
