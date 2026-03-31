package secrets

import (
	"context"
	"os"
)

// EnvStore reads secrets from environment variables.
// No persistence, no encryption. For development and testing.
type EnvStore struct{}

// NewEnvStore creates a new EnvStore.
func NewEnvStore() *EnvStore {
	return &EnvStore{}
}

func (s *EnvStore) Get(_ context.Context, name string) (string, error) {
	return os.Getenv(name), nil
}

func (s *EnvStore) Set(_ context.Context, name, value string) error {
	return os.Setenv(name, value)
}

func (s *EnvStore) Delete(_ context.Context, name string) error {
	return os.Unsetenv(name)
}

func (s *EnvStore) List(_ context.Context) ([]SecretMeta, error) {
	return nil, nil // env vars can't be enumerated meaningfully
}

func (s *EnvStore) Close() error { return nil }
