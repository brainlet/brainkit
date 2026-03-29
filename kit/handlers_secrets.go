package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/kit/secrets"
	"github.com/brainlet/brainkit/sdk/messages"
)

// SecretsDomain handles secrets.set/get/delete/list/rotate bus commands.
type SecretsDomain struct {
	kit *Kernel
}

func newSecretsDomain(k *Kernel) *SecretsDomain {
	return &SecretsDomain{kit: k}
}

func (d *SecretsDomain) Set(ctx context.Context, req messages.SecretsSetMsg) (*messages.SecretsSetResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("secrets.set: name is required")
	}
	if err := d.kit.secretStore.Set(ctx, req.Name, req.Value); err != nil {
		return nil, err
	}

	// Get version from metadata
	version := 1
	metas, _ := d.kit.secretStore.List(ctx)
	for _, m := range metas {
		if m.Name == req.Name {
			version = m.Version
			break
		}
	}

	// Audit event
	d.kit.emitSecretEvent(ctx, messages.SecretsStoredEvent{Name: req.Name, Version: version})

	return &messages.SecretsSetResp{Stored: true, Version: version}, nil
}

func (d *SecretsDomain) Get(ctx context.Context, req messages.SecretsGetMsg) (*messages.SecretsGetResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("secrets.get: name is required")
	}
	val, err := d.kit.secretStore.Get(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// Audit event
	d.kit.emitSecretEvent(ctx, messages.SecretsAccessedEvent{Name: req.Name, Accessor: d.kit.callerID})

	return &messages.SecretsGetResp{Value: val}, nil
}

func (d *SecretsDomain) Delete(ctx context.Context, req messages.SecretsDeleteMsg) (*messages.SecretsDeleteResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("secrets.delete: name is required")
	}
	if err := d.kit.secretStore.Delete(ctx, req.Name); err != nil {
		return nil, err
	}

	// Audit event
	d.kit.emitSecretEvent(ctx, messages.SecretsDeletedEvent{Name: req.Name})

	return &messages.SecretsDeleteResp{Deleted: true}, nil
}

func (d *SecretsDomain) List(ctx context.Context, _ messages.SecretsListMsg) (*messages.SecretsListResp, error) {
	metas, err := d.kit.secretStore.List(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]messages.SecretMetaInfo, 0, len(metas))
	for _, m := range metas {
		infos = append(infos, messages.SecretMetaInfo{
			Name:      m.Name,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
			UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
			Version:   m.Version,
		})
	}
	return &messages.SecretsListResp{Secrets: infos}, nil
}

func (d *SecretsDomain) Rotate(ctx context.Context, req messages.SecretsRotateMsg) (*messages.SecretsRotateResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("secrets.rotate: name is required")
	}

	// 1. Update the secret
	if err := d.kit.secretStore.Set(ctx, req.Name, req.NewValue); err != nil {
		return nil, err
	}

	version := 1
	metas, _ := d.kit.secretStore.List(ctx)
	for _, m := range metas {
		if m.Name == req.Name {
			version = m.Version
			break
		}
	}

	var restartedPlugins []string

	// 2. Restart plugins that reference this secret (if requested)
	if req.Restart && d.kit.node != nil {
		for _, p := range d.kit.node.ListRunningPlugins() {
			if pluginUsesSecret(p, req.Name) {
				if err := d.kit.node.RestartPlugin(ctx, p.Name); err == nil {
					restartedPlugins = append(restartedPlugins, p.Name)
				}
			}
		}
	}

	// 3. If it's a provider key, refresh JS-side cache
	d.kit.refreshProviderIfSecret(req.Name, req.NewValue)

	// 4. Audit event
	d.kit.emitSecretEvent(ctx, messages.SecretsRotatedEvent{
		Name: req.Name, Version: version, RestartedPlugins: restartedPlugins,
	})

	return &messages.SecretsRotateResp{
		Rotated: true, Version: version, RestartedPlugins: restartedPlugins,
	}, nil
}

// pluginUsesSecret checks if a running plugin's env references a secret by name.
func pluginUsesSecret(p RunningPlugin, secretName string) bool {
	for _, v := range p.Config.Env {
		if v == "$secret:"+secretName {
			return true
		}
	}
	return false
}

// emitSecretEvent publishes a secrets audit event.
func (k *Kernel) emitSecretEvent(ctx context.Context, event messages.BrainkitMessage) {
	payload, _ := json.Marshal(event)
	k.publish(ctx, event.BusTopic(), payload)
}

// refreshProviderIfSecret checks if a secret name matches a known provider key pattern
// and refreshes the JS-side provider cache if so.
func (k *Kernel) refreshProviderIfSecret(name, newValue string) {
	// Map secret names to provider env patterns
	providerKeys := map[string]string{
		"OPENAI_API_KEY":    "openai",
		"ANTHROPIC_API_KEY": "anthropic",
		"GOOGLE_API_KEY":    "google",
		"MISTRAL_API_KEY":   "mistral",
		"GROQ_API_KEY":      "groq",
		"DEEPSEEK_API_KEY":  "deepseek",
		"XAI_API_KEY":       "xai",
		"COHERE_API_KEY":    "cohere",
	}
	// TODO: meed a way to add more from configuration, e.g. for custom providers or new ones

	provName, ok := providerKeys[name]
	if !ok {
		return
	}

	// Update the JS-side provider cache
	escaped := secrets.MaskSecret(newValue) // for logging only
	_ = escaped
	k.bridge.Eval("__refresh_provider.js", fmt.Sprintf(
		`if (globalThis.__kit_providers && globalThis.__kit_providers[%q]) {
			globalThis.__kit_providers[%q].APIKey = %q;
			globalThis.__kit_providers[%q].apiKey = %q;
			// Clear provider cache so next model() call re-creates with new key
			delete globalThis.__kit_provider_cache;
		}`, provName, provName, newValue, provName, newValue))
}
