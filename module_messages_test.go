package brainkit_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	registrymod "github.com/brainlet/brainkit/modules/registry"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/stores"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryModuleMessages(t *testing.T) {
	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "module-message-registry-test",
		FSRoot:    t.TempDir(),
		Modules:   []bkmodule.Module{registrymod.New()},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	add, err := registrymsg.CallProviderAdd(kit, ctx, registrymsg.ProviderAddMsg{
		Name:   "demo",
		Type:   "openai",
		Config: json.RawMessage(`{"APIKey":"sk-test"}`),
	})
	require.NoError(t, err)
	require.True(t, add.Added)

	has, err := registrymsg.CallRegistryHas(kit, ctx, registrymsg.RegistryHasMsg{
		Category: "provider",
		Name:     "demo",
	})
	require.NoError(t, err)
	assert.True(t, has.Found)

	var caller bkmodule.RequestCaller = kit.Caller()
	hasViaCaller, err := registrymsg.CallRegistryHasWithCaller(caller, ctx, registrymsg.RegistryHasMsg{
		Category: "provider",
		Name:     "demo",
	})
	require.NoError(t, err)
	assert.True(t, hasViaCaller.Found)

	list, err := registrymsg.CallRegistryList(kit, ctx, registrymsg.RegistryListMsg{Category: "provider"})
	require.NoError(t, err)
	assert.Contains(t, string(list.Items), `"name":"demo"`)

	removed, err := registrymsg.CallProviderRemove(kit, ctx, registrymsg.ProviderRemoveMsg{Name: "demo"})
	require.NoError(t, err)
	assert.True(t, removed.Removed)
}

func TestSecretsModuleMessages(t *testing.T) {
	tmp := t.TempDir()
	store, err := stores.NewSQLite(filepath.Join(tmp, "kit.db"))
	require.NoError(t, err)

	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "module-message-secrets-test",
		FSRoot:    tmp,
		Store:     store,
		SecretKey: "unit-test-key-that-is-32-bytes!!",
		Modules:   []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	set, err := secretmsg.CallSecretsSet(kit, ctx, secretmsg.SecretsSetMsg{Name: "API_KEY", Value: "v1"})
	require.NoError(t, err)
	require.True(t, set.Stored)

	got, err := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "API_KEY"})
	require.NoError(t, err)
	assert.Equal(t, "v1", got.Value)

	rotated, err := secretmsg.CallSecretsRotate(kit, ctx, secretmsg.SecretsRotateMsg{Name: "API_KEY", NewValue: "v2"})
	require.NoError(t, err)
	assert.True(t, rotated.Rotated)

	list, err := secretmsg.CallSecretsList(kit, ctx, secretmsg.SecretsListMsg{})
	require.NoError(t, err)
	require.Len(t, list.Secrets, 1)
	assert.Equal(t, "API_KEY", list.Secrets[0].Name)
	assert.Equal(t, 2, list.Secrets[0].Version)

	deleted, err := secretmsg.CallSecretsDelete(kit, ctx, secretmsg.SecretsDeleteMsg{Name: "API_KEY"})
	require.NoError(t, err)
	assert.True(t, deleted.Deleted)
}
