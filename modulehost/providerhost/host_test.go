package providerhost

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/internal/types"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

func TestAutoDetectProvidersUsesEnvVarsAndHonorsExplicitMap(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "from-os-env")

	cfg := types.KernelConfig{
		EnvVars: map[string]string{"OPENAI_API_KEY": "from-config-env"},
	}
	AutoDetectProviders(&cfg)

	reg, ok := cfg.AIProviders["openai"]
	if !ok {
		t.Fatalf("expected openai provider to be auto-detected")
	}
	creds := ExtractProviderCredentials(reg)
	if creds.APIKey != "from-config-env" {
		t.Fatalf("detected API key = %q, want EnvVars override", creds.APIKey)
	}

	explicit := types.KernelConfig{AIProviders: map[string]types.AIProviderRegistration{}}
	AutoDetectProviders(&explicit)
	if len(explicit.AIProviders) != 0 {
		t.Fatalf("explicit empty provider map must disable auto-detection, got %d providers", len(explicit.AIProviders))
	}
}

func TestManagerRegistersProvidersAndRefreshesMappedSecret(t *testing.T) {
	var called struct {
		fn   string
		args any
	}
	manager, err := NewManager(types.KernelConfig{
		AIProviders: map[string]types.AIProviderRegistration{
			"openai": {
				Type:   types.AIProviderOpenAI,
				Config: types.OpenAIProviderConfig{APIKey: "initial"},
			},
		},
		ProviderKeyMapping: map[string]string{"CUSTOM_KEY": "openai"},
	}, Hooks{
		CallJSSync: func(fn string, args any) {
			called.fn = fn
			called.args = args
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if !manager.Registry().HasAIProvider("openai") {
		t.Fatalf("configured provider was not registered")
	}

	manager.RefreshProviderIfSecret("CUSTOM_KEY", "rotated")

	if called.fn != "__brainkit.secrets.refreshProvider" {
		t.Fatalf("refresh fn = %q", called.fn)
	}
	payload, ok := called.args.(map[string]string)
	if !ok {
		t.Fatalf("refresh args type = %T", called.args)
	}
	if payload["provider"] != "openai" || payload["apiKey"] != "rotated" {
		t.Fatalf("refresh payload = %#v", payload)
	}
}

func TestProbeVectorStoreUsesRuntimeHook(t *testing.T) {
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		EvalTS: func(_ context.Context, filename, code string) (string, error) {
			if filename != "__probe_vectorstore.ts" {
				t.Fatalf("filename = %q", filename)
			}
			return `{"available":true}`, nil
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.Registry().RegisterVectorStore("main", provreg.VectorStoreRegistration{
		Type: provreg.VectorStoreLibSQL,
	}); err != nil {
		t.Fatalf("RegisterVectorStore: %v", err)
	}

	result := manager.ProbeVectorStore("main")
	if !result.Available || result.Error != "" {
		t.Fatalf("probe result = %#v", result)
	}
	stores := manager.Registry().ListVectorStores()
	if len(stores) != 1 || !stores[0].Healthy || stores[0].LastProbed.IsZero() {
		t.Fatalf("registry probe state = %#v", stores)
	}
}
