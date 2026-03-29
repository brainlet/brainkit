package secrets

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptedKVStore stores secrets in a SQL database, encrypted with XChaCha20-Poly1305.
// Master key is derived via Argon2id per-secret (unique salt per entry).
// If masterKey is empty, secrets are stored as base64 (unencrypted dev mode).
type EncryptedKVStore struct {
	db        *sql.DB
	masterKey string
}

// NewEncryptedKVStore creates an encrypted secret store backed by a sql.DB.
// The db should already have the secrets table created (via the schema in store.go).
// If masterKey is empty, secrets are base64-encoded but NOT encrypted (dev mode).
func NewEncryptedKVStore(db *sql.DB, masterKey string) (*EncryptedKVStore, error) {
	// Create table if not exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS secrets (
		name TEXT PRIMARY KEY,
		encrypted_value BLOB NOT NULL,
		salt BLOB NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("secrets: create table: %w", err)
	}
	return &EncryptedKVStore{db: db, masterKey: masterKey}, nil
}

func (s *EncryptedKVStore) Get(_ context.Context, name string) (string, error) {
	var encValue []byte
	var salt []byte
	err := s.db.QueryRow("SELECT encrypted_value, salt FROM secrets WHERE name = ?", name).Scan(&encValue, &salt)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("secrets.get %q: %w", name, err)
	}
	return s.decrypt(encValue, salt)
}

func (s *EncryptedKVStore) Set(_ context.Context, name, value string) error {
	encValue, salt, err := s.encrypt(value)
	if err != nil {
		return fmt.Errorf("secrets.set %q: encrypt: %w", name, err)
	}
	now := time.Now().Format(time.RFC3339)
	_, err = s.db.Exec(`INSERT INTO secrets (name, encrypted_value, salt, version, created_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			encrypted_value = excluded.encrypted_value,
			salt = excluded.salt,
			version = version + 1,
			updated_at = excluded.updated_at`,
		name, encValue, salt, now, now)
	return err
}

func (s *EncryptedKVStore) Delete(_ context.Context, name string) error {
	_, err := s.db.Exec("DELETE FROM secrets WHERE name = ?", name)
	return err
}

func (s *EncryptedKVStore) List(_ context.Context) ([]SecretMeta, error) {
	rows, err := s.db.Query("SELECT name, version, created_at, updated_at FROM secrets ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SecretMeta
	for rows.Next() {
		var m SecretMeta
		var createdStr, updatedStr string
		if err := rows.Scan(&m.Name, &m.Version, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *EncryptedKVStore) Close() error { return nil }

// encrypt returns encrypted_value and salt. If no masterKey, base64-encodes.
func (s *EncryptedKVStore) encrypt(plaintext string) ([]byte, []byte, error) {
	if s.masterKey == "" {
		// Dev mode: base64 encode, empty salt
		return []byte(base64.StdEncoding.EncodeToString([]byte(plaintext))), []byte{}, nil
	}

	// Generate random salt for Argon2id
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("generate salt: %w", err)
	}

	// Derive key via Argon2id
	key := argon2.IDKey([]byte(s.masterKey), salt, 1, 64*1024, 4, chacha20poly1305.KeySize)

	// Encrypt with XChaCha20-Poly1305
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	// ciphertext = nonce + encrypted data
	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, salt, nil
}

// decrypt returns plaintext from encrypted_value and salt.
func (s *EncryptedKVStore) decrypt(encValue, salt []byte) (string, error) {
	if s.masterKey == "" {
		// Dev mode: base64 decode
		decoded, err := base64.StdEncoding.DecodeString(string(encValue))
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
		return string(decoded), nil
	}

	// Derive key via Argon2id (same params as encrypt)
	key := argon2.IDKey([]byte(s.masterKey), salt, 1, 64*1024, 4, chacha20poly1305.KeySize)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	if len(encValue) < aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := encValue[:aead.NonceSize()]
	ciphertext := encValue[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
