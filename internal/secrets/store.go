package secrets

import (
	"context"
	"time"
)

// SecretStore is the pluggable interface for secret management.
// Built-in: EncryptedKVStore (default), EnvStore (dev).
// External: VaultStore (enterprise, separate import).
type SecretStore interface {
	// Get retrieves a secret by name. Returns empty string if not found.
	Get(ctx context.Context, name string) (string, error)

	// Set stores or updates a secret. Overwrites if exists.
	Set(ctx context.Context, name, value string) error

	// Delete removes a secret.
	Delete(ctx context.Context, name string) error

	// List returns metadata for all secrets (names, timestamps — NOT values).
	List(ctx context.Context) ([]SecretMeta, error)

	// Close releases resources.
	Close() error
}

// SecretMeta describes a secret without its value.
type SecretMeta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   int       `json:"version"` // increments on each Set
}
