package engine

import (
	"context"
	"database/sql"
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
	return k.close(ctx)
}

// Close shuts down with a short drain timeout (5s).
func (k *Kernel) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return k.Shutdown(ctx)
}

// resolveSecretStore determines the secret store from config with clear precedence:
// 1. Explicit SecretStore → use it
// 2. Store exposing DB() *sql.DB + SecretKey -> encrypted KV store
// 3. Store exposing DB() *sql.DB + no SecretKey -> unencrypted KV store (dev mode, logged warning)
// 4. No SQL-backed Store -> environment variable fallback
func resolveSecretStore(cfg types.KernelConfig, logger *slog.Logger) secrets.SecretStore {
	if cfg.SecretStore != nil {
		return cfg.SecretStore
	}

	key := cfg.SecretKey
	if key == "" {
		key = os.Getenv("BRAINKIT_SECRET_KEY")
	}

	type sqlDBStore interface {
		DB() *sql.DB
	}
	sqlStore, hasSQL := cfg.Store.(sqlDBStore)
	if !hasSQL || sqlStore == nil || sqlStore.DB() == nil {
		return secrets.NewEnvStore()
	}

	if key == "" {
		logger.Warn("SecretKey not set, secrets stored without encryption")
	}

	store, err := secrets.NewEncryptedKVStore(sqlStore.DB(), key)
	if err != nil {
		types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "CreateEncryptedSecretStore", Cause: err,
		}, types.ErrorContext{Operation: "CreateEncryptedSecretStore", Component: "kernel"})
		return secrets.NewEnvStore()
	}
	return store
}

// close is the internal shutdown logic.
func (k *Kernel) close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	k.closeMu.Lock()
	defer k.closeMu.Unlock()

	k.mu.Lock()
	if k.closeComplete {
		k.mu.Unlock()
		return nil
	}
	k.closed = true
	k.mu.Unlock()

	if k.shutdownCancel != nil {
		k.shutdownCancel()
	}

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Stop all stream heartbeat goroutines
	if k.streamTracker != nil {
		collect(k.streamTracker.CloseAll(ctx))
	}

	if k.jsRuntime != nil {
		k.jsRuntime.Interrupt()
	}

	collect(k.waitBackground(ctx))

	// Shut down router first (stops processing messages)
	if k.runtimeHost != nil {
		collect(k.runtimeHost.Close())
	}
	if k.transportHost != nil {
		collect(k.transportHost.CloseRouter())
	}

	// Close the Caller — unsubscribes inbox, finalizes pending with
	// ErrCallerClosed.
	if k.transportHost != nil {
		collect(k.transportHost.CloseCallerContext(ctx))
	}

	if rt := k.jsRuntime; rt != nil {
		collect(rt.Shutdown(ctx))
		k.DetachJSRuntime(rt)
		k.SetRuntimeConfigJSRuntime(false)
	}
	if k.providerHost != nil {
		collect(k.providerHost.CloseContext(ctx))
	}
	if k.config.Store != nil {
		collect(k.config.Store.Close())
	}
	if k.storageHost != nil {
		collect(k.storageHost.CloseAllContext(ctx))
	}

	// Shut down transport last (only if we own it — Node owns its own)
	if k.transportHost != nil {
		collect(k.transportHost.CloseOwnedTransport())
	}

	if firstErr == nil {
		k.mu.Lock()
		k.closeComplete = true
		k.mu.Unlock()
	}
	return firstErr
}
