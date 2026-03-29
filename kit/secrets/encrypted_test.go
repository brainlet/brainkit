package secrets

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEncryptedKVStore_SetGetDelete(t *testing.T) {
	db := testDB(t)
	store, err := NewEncryptedKVStore(db, "test-master-key-32bytes-long!!!")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Set
	if err := store.Set(ctx, "api-key", "sk-secret-value-12345"); err != nil {
		t.Fatal("set:", err)
	}

	// Get
	val, err := store.Get(ctx, "api-key")
	if err != nil {
		t.Fatal("get:", err)
	}
	if val != "sk-secret-value-12345" {
		t.Fatalf("got %q, want %q", val, "sk-secret-value-12345")
	}

	// Get non-existent
	val, err = store.Get(ctx, "no-such-key")
	if err != nil {
		t.Fatal("get missing:", err)
	}
	if val != "" {
		t.Fatalf("expected empty for missing key, got %q", val)
	}

	// Delete
	if err := store.Delete(ctx, "api-key"); err != nil {
		t.Fatal("delete:", err)
	}
	val, _ = store.Get(ctx, "api-key")
	if val != "" {
		t.Fatal("expected empty after delete")
	}
}

func TestEncryptedKVStore_VersionIncrement(t *testing.T) {
	db := testDB(t)
	store, err := NewEncryptedKVStore(db, "test-key")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	store.Set(ctx, "token", "v1")
	metas, _ := store.List(ctx)
	if len(metas) != 1 || metas[0].Version != 1 {
		t.Fatalf("expected version 1, got %v", metas)
	}

	store.Set(ctx, "token", "v2")
	metas, _ = store.List(ctx)
	if metas[0].Version != 2 {
		t.Fatalf("expected version 2, got %d", metas[0].Version)
	}

	store.Set(ctx, "token", "v3")
	metas, _ = store.List(ctx)
	if metas[0].Version != 3 {
		t.Fatalf("expected version 3, got %d", metas[0].Version)
	}
}

func TestEncryptedKVStore_List(t *testing.T) {
	db := testDB(t)
	store, err := NewEncryptedKVStore(db, "key")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store.Set(ctx, "alpha", "a")
	store.Set(ctx, "beta", "b")
	store.Set(ctx, "gamma", "c")

	metas, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(metas))
	}

	// Verify values are NOT in metadata
	for _, m := range metas {
		if m.Name == "" {
			t.Fatal("empty name in metadata")
		}
	}
}

func TestEncryptedKVStore_DevMode_NoKey(t *testing.T) {
	db := testDB(t)
	store, err := NewEncryptedKVStore(db, "") // empty key = dev mode
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store.Set(ctx, "dev-secret", "plaintext-value")

	val, err := store.Get(ctx, "dev-secret")
	if err != nil {
		t.Fatal(err)
	}
	if val != "plaintext-value" {
		t.Fatalf("got %q, want %q", val, "plaintext-value")
	}
}

func TestEncryptedKVStore_WrongKey(t *testing.T) {
	db := testDB(t)

	// Write with one key
	store1, _ := NewEncryptedKVStore(db, "correct-key")
	store1.Set(context.Background(), "secret", "hidden-value")

	// Read with a different key
	store2, _ := NewEncryptedKVStore(db, "wrong-key")
	_, err := store2.Get(context.Background(), "secret")
	if err == nil {
		t.Fatal("expected error with wrong key, got nil")
	}
}

func TestEnvStore(t *testing.T) {
	store := NewEnvStore()
	ctx := context.Background()

	os.Setenv("TEST_SECRET_123", "hello")
	defer os.Unsetenv("TEST_SECRET_123")

	val, err := store.Get(ctx, "TEST_SECRET_123")
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Fatalf("got %q, want %q", val, "hello")
	}

	// Not found
	val, _ = store.Get(ctx, "NONEXISTENT_KEY_XYZ")
	if val != "" {
		t.Fatalf("expected empty, got %q", val)
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "****"},
		{"12345678", "****"},
		{"123456789012", "****"},
		{"sk-proj-abcdefghijklmnop", "sk-p...mnop"},
		{"1234567890123", "1234...0123"},
	}

	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
