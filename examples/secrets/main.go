// Command secrets demonstrates the encrypted secret store lifecycle through
// the modules/secrets typed bus messages. Uses an explicit SecretKey so values
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
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/stores"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("secrets: %v", err)
	}
}

func run() error {
	tmp := mustTempDir()
	defer cleanupTemp(tmp)

	store, err := stores.NewSQLite(filepath.Join(tmp, "kit.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "secrets-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Store:     store,
		SecretKey: "demo-secret-key-sufficiently-long!",
		Modules:   []brainkit.Module{secretsmod.New()},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set.
	if _, err := secretmsg.CallSecretsSet(kit, ctx, secretmsg.SecretsSetMsg{Name: "API_KEY", Value: "sk-demo-v1"}); err != nil {
		return fmt.Errorf("set API_KEY: %w", err)
	}
	fmt.Println("set API_KEY = sk-demo-v1")

	// Get.
	got, err := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "API_KEY"})
	if err != nil {
		return fmt.Errorf("get API_KEY: %w", err)
	}
	fmt.Printf("get API_KEY -> %s\n", got.Value)

	// Rotate.
	if _, err := secretmsg.CallSecretsRotate(kit, ctx, secretmsg.SecretsRotateMsg{Name: "API_KEY", NewValue: "sk-demo-v2"}); err != nil {
		return fmt.Errorf("rotate API_KEY: %w", err)
	}
	rotated, _ := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "API_KEY"})
	fmt.Printf("rotate API_KEY -> %s\n", rotated.Value)

	// List.
	list, err := secretmsg.CallSecretsList(kit, ctx, secretmsg.SecretsListMsg{})
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}
	fmt.Printf("list: %d secret(s)\n", len(list.Secrets))
	for _, m := range list.Secrets {
		fmt.Printf("  %s (version=%d, updated=%s)\n", m.Name, m.Version, m.UpdatedAt)
	}

	// Delete.
	if _, err := secretmsg.CallSecretsDelete(kit, ctx, secretmsg.SecretsDeleteMsg{Name: "API_KEY"}); err != nil {
		return fmt.Errorf("delete API_KEY: %w", err)
	}
	after, _ := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "API_KEY"})
	fmt.Printf("delete API_KEY -> Get returns %q\n", after.Value)

	return nil
}

func mustTempDir() string {
	dir, err := tempDir()
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	return dir
}
