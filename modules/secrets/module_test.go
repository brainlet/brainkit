package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	plugincap "github.com/brainlet/brainkit/modulecap/plugin"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
)

func TestRotatePropagatesProviderSecretRefreshError(t *testing.T) {
	want := errors.New("provider cache refresh failed")
	store := newMemorySecretStore()
	module := &Module{
		store:                 store,
		refreshProviderSecret: &failingProviderSecretRefresher{err: want},
	}

	ctx := context.WithValue(context.Background(), rotateContextKey{}, "rotate")
	_, err := module.Rotate(ctx, secretmsg.SecretsRotateMsg{Name: "OPENAI_API_KEY", NewValue: "rotated"})
	if !errors.Is(err, want) {
		t.Fatalf("Rotate error = %v, want %v", err, want)
	}

	got, err := store.Get(context.Background(), "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("Get rotated secret: %v", err)
	}
	if got != "rotated" {
		t.Fatalf("rotated secret = %q, want rotated", got)
	}
}

func TestFactoryDescriptorUsesTypedProviderSecretRefresher(t *testing.T) {
	desc := bkmodule.NormalizeDescriptor("secrets", Factory{}.Describe())
	var found bool
	for _, cap := range desc.Capabilities {
		if cap.Name != bkmodule.CapabilityRefreshProviderSecret {
			continue
		}
		found = true
		if cap.Direction != bkmodule.CapabilityOptional {
			t.Fatalf("refresh provider capability direction = %s, want optional", cap.Direction)
		}
		if cap.Type == "func(string, string)" {
			t.Fatalf("refresh provider capability still advertises raw callback type")
		}
	}
	if !found {
		t.Fatal("refresh provider capability descriptor missing")
	}
}

func TestRotateReturnsPluginRestartFailures(t *testing.T) {
	want := errors.New("restart failed")
	store := newMemorySecretStore()
	restarter := &fakePluginRestarter{
		plugins: []types.RunningPlugin{
			{Name: "ok", Config: types.PluginConfig{Env: map[string]string{"TOKEN": "$secret:BOT_TOKEN"}}},
			{Name: "bad", Config: types.PluginConfig{Env: map[string]string{"TOKEN": "$secret:BOT_TOKEN"}}},
			{Name: "skip", Config: types.PluginConfig{Env: map[string]string{"TOKEN": "$secret:OTHER_TOKEN"}}},
		},
		failures: map[string]error{"bad": want},
	}
	module := &Module{
		store:           store,
		pluginRestarter: func() plugincap.Restarter { return restarter },
	}

	_, err := module.Rotate(context.Background(), secretmsg.SecretsRotateMsg{
		Name:     "BOT_TOKEN",
		NewValue: "rotated",
		Restart:  true,
	})
	if !errors.Is(err, want) {
		t.Fatalf("Rotate error = %v, want %v", err, want)
	}
	if err == nil || !containsAll(err.Error(), "bad", "BOT_TOKEN") {
		t.Fatalf("Rotate error %q should include failed plugin and secret name", err)
	}
	if got := restarter.restarted; len(got) != 2 || got[0] != "ok" || got[1] != "bad" {
		t.Fatalf("restarted plugins = %v, want [ok bad]", got)
	}
	value, err := store.Get(context.Background(), "BOT_TOKEN")
	if err != nil {
		t.Fatalf("Get rotated secret: %v", err)
	}
	if value != "rotated" {
		t.Fatalf("rotated secret = %q, want rotated", value)
	}
}

type rotateContextKey struct{}

type failingProviderSecretRefresher struct {
	err error
}

func (f *failingProviderSecretRefresher) RefreshProviderSecret(context.Context, string, string) error {
	return f.err
}

type fakePluginRestarter struct {
	plugins   []types.RunningPlugin
	failures  map[string]error
	restarted []string
}

func (r *fakePluginRestarter) ListRunningPlugins() []types.RunningPlugin {
	return r.plugins
}

func (r *fakePluginRestarter) RestartPlugin(_ context.Context, name string) error {
	r.restarted = append(r.restarted, name)
	return r.failures[name]
}

func containsAll(s string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}

type memorySecretStore struct {
	values   map[string]string
	versions map[string]int
}

func newMemorySecretStore() *memorySecretStore {
	return &memorySecretStore{
		values:   map[string]string{},
		versions: map[string]int{},
	}
}

func (s *memorySecretStore) Get(_ context.Context, name string) (string, error) {
	return s.values[name], nil
}

func (s *memorySecretStore) Set(_ context.Context, name, value string) error {
	s.values[name] = value
	s.versions[name]++
	return nil
}

func (s *memorySecretStore) Delete(_ context.Context, name string) error {
	delete(s.values, name)
	delete(s.versions, name)
	return nil
}

func (s *memorySecretStore) List(context.Context) ([]types.SecretMeta, error) {
	now := time.Now()
	metas := make([]types.SecretMeta, 0, len(s.values))
	for name := range s.values {
		metas = append(metas, types.SecretMeta{
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   s.versions[name],
		})
	}
	return metas, nil
}

func (s *memorySecretStore) Close() error { return nil }
