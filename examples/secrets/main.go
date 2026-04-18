// Command secrets demonstrates the encrypted secret store
// lifecycle: Set / Get / Rotate / Delete through the
// Kit.Secrets() accessor. Uses an explicit SecretKey so values
// are encrypted at rest.
//
// Run from the repo root:
//
//	go run ./examples/secrets
package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("secrets: %v", err)
	}
}

func run() error {
	tmp := mustTempDir()
	defer cleanupTemp(tmp)

	store, err := brainkit.NewSQLiteStore(filepath.Join(tmp, "kit.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "secrets-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Store:     store,
		SecretKey: "demo-secret-key-sufficiently-long!",
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secrets := kit.Secrets()

	// Set.
	if err := secrets.Set(ctx, "API_KEY", "sk-demo-v1"); err != nil {
		return fmt.Errorf("set API_KEY: %w", err)
	}
	fmt.Println("set API_KEY = sk-demo-v1")

	// Get.
	v, err := secrets.Get(ctx, "API_KEY")
	if err != nil {
		return fmt.Errorf("get API_KEY: %w", err)
	}
	fmt.Printf("get API_KEY → %s\n", v)

	// Rotate.
	if err := secrets.Rotate(ctx, "API_KEY", "sk-demo-v2"); err != nil {
		return fmt.Errorf("rotate API_KEY: %w", err)
	}
	v, _ = secrets.Get(ctx, "API_KEY")
	fmt.Printf("rotate API_KEY → %s\n", v)

	// List.
	meta, err := secrets.List(ctx)
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}
	fmt.Printf("list: %d secret(s)\n", len(meta))
	for _, m := range meta {
		fmt.Printf("  %s (version=%d, updated=%s)\n", m.Name, m.Version, m.UpdatedAt.Format(time.RFC3339))
	}

	// Delete.
	if err := secrets.Delete(ctx, "API_KEY"); err != nil {
		return fmt.Errorf("delete API_KEY: %w", err)
	}
	after, _ := secrets.Get(ctx, "API_KEY")
	fmt.Printf("delete API_KEY → Get returns %q\n", after)

	return nil
}

func mustTempDir() string {
	dir, err := tempDir()
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	return dir
}
