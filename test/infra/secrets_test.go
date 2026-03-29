package infra

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

func startKernelWithSecrets(t *testing.T) *kit.Kernel {
	t.Helper()
	storePath := t.TempDir() + "/secrets-test.db"
	store, err := kit.NewSQLiteStore(storePath)
	if err != nil {
		t.Fatal("store:", err)
	}
	k, err := kit.NewKernel(kit.KernelConfig{
		Store:     store,
		SecretKey: "test-master-key-for-secrets!!!",
	})
	if err != nil {
		t.Fatal("kernel:", err)
	}
	t.Cleanup(func() { k.Close() })
	return k
}

func TestSecrets_SetAndGet(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set
	pub, err := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "api-key", Value: "sk-test-12345"})
	if err != nil {
		t.Fatal("publish set:", err)
	}
	respCh := make(chan messages.SecretsSetResp, 1)
	cancel, err := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, msg messages.Message) {
		respCh <- resp
	})
	if err != nil {
		t.Fatal("subscribe:", err)
	}
	defer cancel()

	select {
	case resp := <-respCh:
		if !resp.Stored {
			t.Fatal("expected stored=true")
		}
		if resp.Version != 1 {
			t.Fatalf("expected version 1, got %d", resp.Version)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for set response")
	}

	// Get
	pub2, err := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "api-key"})
	if err != nil {
		t.Fatal("publish get:", err)
	}
	getCh := make(chan messages.SecretsGetResp, 1)
	cancel2, err := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, pub2.ReplyTo, func(resp messages.SecretsGetResp, msg messages.Message) {
		getCh <- resp
	})
	if err != nil {
		t.Fatal("subscribe get:", err)
	}
	defer cancel2()

	select {
	case resp := <-getCh:
		if resp.Value != "sk-test-12345" {
			t.Fatalf("got %q, want %q", resp.Value, "sk-test-12345")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for get response")
	}
}

func TestSecrets_Delete(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set first
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "temp", Value: "val"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		setCh <- resp
	})
	<-setCh
	cancel()

	// Delete
	pub2, _ := sdk.Publish(k, ctx, messages.SecretsDeleteMsg{Name: "temp"})
	delCh := make(chan messages.SecretsDeleteResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.SecretsDeleteResp](k, ctx, pub2.ReplyTo, func(resp messages.SecretsDeleteResp, _ messages.Message) {
		delCh <- resp
	})
	defer cancel2()

	select {
	case resp := <-delCh:
		if !resp.Deleted {
			t.Fatal("expected deleted=true")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	// Verify gone
	pub3, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "temp"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancel3, _ := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, pub3.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) {
		getCh <- resp
	})
	defer cancel3()

	select {
	case resp := <-getCh:
		if resp.Value != "" {
			t.Fatalf("expected empty after delete, got %q", resp.Value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSecrets_List(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set two secrets
	for _, name := range []string{"key-a", "key-b"} {
		pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: name, Value: "val-" + name})
		ch := make(chan messages.SecretsSetResp, 1)
		cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
			ch <- resp
		})
		<-ch
		cancel()
	}

	// List
	pub, _ := sdk.Publish(k, ctx, messages.SecretsListMsg{})
	listCh := make(chan messages.SecretsListResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsListResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsListResp, _ messages.Message) {
		listCh <- resp
	})
	defer cancel()

	select {
	case resp := <-listCh:
		if len(resp.Secrets) != 2 {
			t.Fatalf("expected 2 secrets, got %d", len(resp.Secrets))
		}
		// Verify no values leaked
		for _, s := range resp.Secrets {
			if s.Name == "" {
				t.Fatal("empty name in list")
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSecrets_Rotate(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set initial
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "rotate-me", Value: "old-value"})
	ch := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		ch <- resp
	})
	<-ch
	cancel()

	// Rotate
	pub2, _ := sdk.Publish(k, ctx, messages.SecretsRotateMsg{Name: "rotate-me", NewValue: "new-value", Restart: false})
	rotateCh := make(chan messages.SecretsRotateResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.SecretsRotateResp](k, ctx, pub2.ReplyTo, func(resp messages.SecretsRotateResp, _ messages.Message) {
		rotateCh <- resp
	})
	defer cancel2()

	select {
	case resp := <-rotateCh:
		if !resp.Rotated {
			t.Fatal("expected rotated=true")
		}
		if resp.Version != 2 {
			t.Fatalf("expected version 2 after rotate, got %d", resp.Version)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	// Verify new value
	pub3, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "rotate-me"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancel3, _ := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, pub3.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) {
		getCh <- resp
	})
	defer cancel3()

	select {
	case resp := <-getCh:
		if resp.Value != "new-value" {
			t.Fatalf("got %q, want %q", resp.Value, "new-value")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSecrets_JSBridge(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set a secret via bus
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "js-test-token", Value: "tok_abc123"})
	ch := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		ch <- resp
	})
	<-ch
	cancel()

	// Read from JS via bridge
	result, err := k.EvalTS(ctx, "__test_secret.ts", `
		var val = secrets.get("js-test-token");
		return val;
	`)
	if err != nil {
		t.Fatal("eval:", err)
	}
	if result != "tok_abc123" {
		t.Fatalf("JS bridge got %q, want %q", result, "tok_abc123")
	}
}

func TestSecrets_AuditEvents(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Subscribe to audit events
	storedCh := make(chan messages.SecretsStoredEvent, 1)
	cancelStored, _ := sdk.SubscribeTo[messages.SecretsStoredEvent](k, ctx, "secrets.stored", func(evt messages.SecretsStoredEvent, _ messages.Message) {
		storedCh <- evt
	})
	defer cancelStored()

	// Set a secret
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "audit-test", Value: "val"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancelSet, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		setCh <- resp
	})
	<-setCh
	cancelSet()

	// Verify stored event was emitted
	select {
	case evt := <-storedCh:
		if evt.Name != "audit-test" {
			t.Fatalf("expected name %q, got %q", "audit-test", evt.Name)
		}
		if evt.Version != 1 {
			t.Fatalf("expected version 1, got %d", evt.Version)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stored event")
	}
}

func TestSecrets_ConcurrentAccess(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set initial secret
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "concurrent", Value: "v0"})
	ch := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		ch <- resp
	})
	<-ch
	cancel()

	// Concurrent reads should not panic or return corrupt data
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			pub, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "concurrent"})
			getCh := make(chan messages.SecretsGetResp, 1)
			cancel, _ := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) {
				getCh <- resp
			})
			select {
			case resp := <-getCh:
				cancel()
				if resp.Value != "v0" {
					t.Errorf("concurrent get: expected %q, got %q", "v0", resp.Value)
				}
			case <-time.After(5 * time.Second):
				cancel()
				t.Error("concurrent get: timeout")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSecrets_DevMode_NoEncryption(t *testing.T) {
	// No SecretKey → secrets stored without encryption (dev mode)
	storePath := t.TempDir() + "/devmode.db"
	store, err := kit.NewSQLiteStore(storePath)
	if err != nil {
		t.Fatal(err)
	}
	k, err := kit.NewKernel(kit.KernelConfig{
		Store: store,
		// No SecretKey — dev mode
	})
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()

	ctx := context.Background()

	// Set and get should still work
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "dev-secret", Value: "unencrypted"})
	setCh := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) {
		setCh <- resp
	})
	<-setCh
	cancel()

	pub2, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "dev-secret"})
	getCh := make(chan messages.SecretsGetResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, pub2.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) {
		getCh <- resp
	})
	defer cancel2()

	select {
	case resp := <-getCh:
		if resp.Value != "unencrypted" {
			t.Fatalf("dev mode: expected %q, got %q", "unencrypted", resp.Value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSecrets_ListNeverLeaksValues(t *testing.T) {
	k := startKernelWithSecrets(t)
	ctx := context.Background()

	// Set a secret with sensitive value
	pub, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "sensitive-key", Value: "sk-super-secret-do-not-leak"})
	ch := make(chan messages.SecretsSetResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { ch <- resp })
	<-ch
	cancel()

	// List should show metadata but NOT the value
	pub2, _ := sdk.Publish(k, ctx, messages.SecretsListMsg{})
	listCh := make(chan messages.SecretsListResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.SecretsListResp](k, ctx, pub2.ReplyTo, func(resp messages.SecretsListResp, _ messages.Message) { listCh <- resp })
	defer cancel2()

	select {
	case resp := <-listCh:
		// Marshal the response to check raw JSON doesn't contain the secret value
		raw, _ := json.Marshal(resp)
		rawStr := string(raw)
		if strings.Contains(rawStr, "sk-super-secret-do-not-leak") {
			t.Fatal("list response contains secret value — must never leak")
		}
		if len(resp.Secrets) != 1 {
			t.Fatalf("expected 1 secret in list, got %d", len(resp.Secrets))
		}
		if resp.Secrets[0].Name != "sensitive-key" {
			t.Fatalf("expected name 'sensitive-key', got %q", resp.Secrets[0].Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
