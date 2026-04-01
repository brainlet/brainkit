package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretsMatrix_SetGetDeleteList — full lifecycle.
func TestSecretsMatrix_SetGetDeleteList(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "test-key-for-list-test-32ch!",
	})
	require.NoError(t, err)
	defer k.Close()
	tk := &testutil.TestKernel{Kernel: k}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set
	pr1, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "lifecycle-key", Value: "lifecycle-val"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	p1 := <-ch1
	unsub1()
	assert.Contains(t, string(p1), "stored")

	// Get
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "lifecycle-key"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	var getResp struct{ Value string `json:"value"` }
	json.Unmarshal(p2, &getResp)
	assert.Equal(t, "lifecycle-val", getResp.Value)

	// List
	pr3, _ := sdk.Publish(tk, ctx, messages.SecretsListMsg{})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	p3 := <-ch3
	unsub3()
	assert.Contains(t, string(p3), "lifecycle-key")

	// Delete
	pr4, _ := sdk.Publish(tk, ctx, messages.SecretsDeleteMsg{Name: "lifecycle-key"})
	ch4 := make(chan []byte, 1)
	unsub4, _ := tk.SubscribeRaw(ctx, pr4.ReplyTo, func(m messages.Message) { ch4 <- m.Payload })
	p4 := <-ch4
	unsub4()
	assert.Contains(t, string(p4), "deleted")

	// Get after delete — should be empty
	pr5, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "lifecycle-key"})
	ch5 := make(chan []byte, 1)
	unsub5, _ := tk.SubscribeRaw(ctx, pr5.ReplyTo, func(m messages.Message) { ch5 <- m.Payload })
	defer unsub5()
	p5 := <-ch5
	var getResp2 struct{ Value string `json:"value"` }
	json.Unmarshal(p5, &getResp2)
	assert.Empty(t, getResp2.Value)
}

// TestSecretsMatrix_Rotate — set then rotate, verify version increments.
func TestSecretsMatrix_Rotate(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set v1
	pr1, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "rot-key", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Rotate to v2
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsRotateMsg{Name: "rot-key", NewValue: "v2"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	assert.Contains(t, string(p2), "rotated")

	// Get — should be v2
	pr3, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "rot-key"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()
	p3 := <-ch3
	var resp struct{ Value string `json:"value"` }
	json.Unmarshal(p3, &resp)
	assert.Equal(t, "v2", resp.Value)
}

// TestSecretsMatrix_ManySecrets — set 20 secrets, list them all.
func TestSecretsMatrix_ManySecrets(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "bulk.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "bulk-test-key-32-characters!!",
	})
	require.NoError(t, err)
	defer k.Close()
	tk := &testutil.TestKernel{Kernel: k}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for i := 0; i < 20; i++ {
		pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{
			Name: fmt.Sprintf("bulk-key-%d", i), Value: fmt.Sprintf("val-%d", i),
		})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		<-ch
		unsub()
	}

	pr, _ := sdk.Publish(tk, ctx, messages.SecretsListMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	p := <-ch

	for i := 0; i < 20; i++ {
		assert.Contains(t, string(p), fmt.Sprintf("bulk-key-%d", i))
	}
}

// TestSecretsMatrix_EncryptedPersistence — secrets survive restart with encryption.
func TestSecretsMatrix_EncryptedPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")
	masterKey := "test-encryption-key-32chars!!"

	// Phase 1: Set encrypted secret
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: masterKey,
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr1, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "enc-key", Value: "enc-secret-val"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := k1.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()
	k1.Close()

	// Phase 2: Reopen with same key, retrieve
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: masterKey,
	})
	require.NoError(t, err)
	defer k2.Close()

	pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "enc-key"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := k2.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()
	p2 := <-ch2
	var resp struct{ Value string `json:"value"` }
	json.Unmarshal(p2, &resp)
	assert.Equal(t, "enc-secret-val", resp.Value)
}

// TestSecretsMatrix_WrongKeyCannotDecrypt — wrong master key fails to decrypt.
func TestSecretsMatrix_WrongKeyCannotDecrypt(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "secrets.db")

	// Set with key A
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr1, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "protected", Value: "sensitive"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := k1.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()
	k1.Close()

	// Reopen with WRONG key
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: "wrong-key-32-characters-long-!",
	})
	require.NoError(t, err)
	defer k2.Close()

	pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "protected"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := k2.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p2 := <-ch2:
		var resp struct {
			Value string `json:"value"`
			Error string `json:"error"`
		}
		json.Unmarshal(p2, &resp)
		// Should either return empty/error or garbage (wrong key = bad decrypt)
		assert.NotEqual(t, "sensitive", resp.Value, "wrong key should not decrypt correctly")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// TestSecretsMatrix_AuditEvents — secrets operations emit audit events.
func TestSecretsMatrix_AuditEvents(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var events []string
	topics := []string{"secrets.stored", "secrets.accessed", "secrets.deleted"}
	var unsubs []func()
	for _, topic := range topics {
		topic := topic
		unsub, _ := tk.SubscribeRaw(ctx, topic, func(m messages.Message) {
			events = append(events, topic)
		})
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	// Set — triggers secrets.stored
	pr1, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "audit-key", Value: "v"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Get — triggers secrets.accessed
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "audit-key"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	// Delete — triggers secrets.deleted
	pr3, _ := sdk.Publish(tk, ctx, messages.SecretsDeleteMsg{Name: "audit-key"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	<-ch3
	unsub3()

	time.Sleep(300 * time.Millisecond)

	assert.Contains(t, events, "secrets.stored")
	assert.Contains(t, events, "secrets.accessed")
	assert.Contains(t, events, "secrets.deleted")
}

// TestSecretsMatrix_FromTS — secrets from .ts surface.
func TestSecretsMatrix_FromTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set via bus first
	pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "ts-secret", Value: "ts-value"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// Read from .ts
	_, err := tk.Deploy(ctx, "secret-read.ts", `
		var val = secrets.get("ts-secret");
		output({value: val, found: val.length > 0});
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__sec.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "ts-value")
	assert.Contains(t, result, `"found":true`)
}
