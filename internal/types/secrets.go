package types

import (
	"context"
	"time"
)

// SecretStore is the pluggable interface for secret management.
// Built-in: EncryptedKVStore (default), EnvStore (dev).
// External: VaultStore (enterprise, separate import).
type SecretStore interface {
	Get(ctx context.Context, name string) (string, error)
	Set(ctx context.Context, name, value string) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]SecretMeta, error)
	Close() error
}

// SecretMeta describes a secret without its value.
type SecretMeta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   int       `json:"version"`
}
