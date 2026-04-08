package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/sdk"
)

// SecretsDomain handles secrets.set/get/delete/list/rotate bus commands.
type SecretsDomain struct {
	store           secrets.SecretStore
	bus             BusPublisher
	callerID        string
	pluginRestarter PluginRestarter    // nil on standalone Kernel
	providerRefresh func(string, string) // refreshProviderIfSecret
}

func newSecretsDomain(store secrets.SecretStore, bus BusPublisher, callerID string, restarter PluginRestarter, providerRefresh func(string, string)) *SecretsDomain {
	return &SecretsDomain{store: store, bus: bus, callerID: callerID, pluginRestarter: restarter, providerRefresh: providerRefresh}
}

// emitSecretEvent publishes a secrets audit event.
func (d *SecretsDomain) emitSecretEvent(ctx context.Context, event sdk.BrainkitMessage) {
	payload, _ := json.Marshal(event)
	d.bus.PublishRaw(ctx, event.BusTopic(), payload)
}

func (d *SecretsDomain) Set(ctx context.Context, req sdk.SecretsSetMsg) (*sdk.SecretsSetResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := d.store.Set(ctx, req.Name, req.Value); err != nil {
		return nil, err
	}

	// Get version from metadata
	version := 1
	metas, _ := d.store.List(ctx)
	for _, m := range metas {
		if m.Name == req.Name {
			version = m.Version
			break
		}
	}

	// Audit event
	d.emitSecretEvent(ctx, sdk.SecretsStoredEvent{Name: req.Name, Version: version, Timestamp: time.Now().Format(time.RFC3339)})

	return &sdk.SecretsSetResp{Stored: true, Version: version}, nil
}

func (d *SecretsDomain) Get(ctx context.Context, req sdk.SecretsGetMsg) (*sdk.SecretsGetResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	val, err := d.store.Get(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// Audit event
	d.emitSecretEvent(ctx, sdk.SecretsAccessedEvent{Name: req.Name, Accessor: d.callerID, Timestamp: time.Now().Format(time.RFC3339)})

	return &sdk.SecretsGetResp{Value: val}, nil
}

func (d *SecretsDomain) Delete(ctx context.Context, req sdk.SecretsDeleteMsg) (*sdk.SecretsDeleteResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := d.store.Delete(ctx, req.Name); err != nil {
		return nil, err
	}

	// Audit event
	d.emitSecretEvent(ctx, sdk.SecretsDeletedEvent{Name: req.Name, Timestamp: time.Now().Format(time.RFC3339)})

	return &sdk.SecretsDeleteResp{Deleted: true}, nil
}

func (d *SecretsDomain) List(ctx context.Context, _ sdk.SecretsListMsg) (*sdk.SecretsListResp, error) {
	metas, err := d.store.List(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]sdk.SecretMetaInfo, 0, len(metas))
	for _, m := range metas {
		infos = append(infos, sdk.SecretMetaInfo{
			Name:      m.Name,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
			UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
			Version:   m.Version,
		})
	}
	return &sdk.SecretsListResp{Secrets: infos}, nil
}

func (d *SecretsDomain) Rotate(ctx context.Context, req sdk.SecretsRotateMsg) (*sdk.SecretsRotateResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}

	// 1. Update the secret
	if err := d.store.Set(ctx, req.Name, req.NewValue); err != nil {
		return nil, err
	}

	version := 1
	metas, _ := d.store.List(ctx)
	for _, m := range metas {
		if m.Name == req.Name {
			version = m.Version
			break
		}
	}

	var restartedPlugins []string

	// 2. Restart plugins that reference this secret (if requested)
	if req.Restart && d.pluginRestarter != nil {
		for _, p := range d.pluginRestarter.ListRunningPlugins() {
			if pluginUsesSecret(p, req.Name) {
				if err := d.pluginRestarter.RestartPlugin(ctx, p.Name); err == nil {
					restartedPlugins = append(restartedPlugins, p.Name)
				}
			}
		}
	}

	// 3. If it's a provider key, refresh JS-side cache
	if d.providerRefresh != nil {
		d.providerRefresh(req.Name, req.NewValue)
	}

	// 4. Audit event
	d.emitSecretEvent(ctx, sdk.SecretsRotatedEvent{
		Name: req.Name, Version: version, RestartedPlugins: restartedPlugins,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	return &sdk.SecretsRotateResp{
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
func (k *Kernel) emitSecretEvent(ctx context.Context, event sdk.BrainkitMessage) {
	payload, _ := json.Marshal(event)
	k.publish(ctx, event.BusTopic(), payload)
}

// defaultProviderKeyMapping is the built-in mapping from secret names to AI provider names.
// Used by refreshProviderIfSecret when KernelConfig.ProviderKeyMapping is nil.
var defaultProviderKeyMapping = map[string]string{
	"OPENAI_API_KEY":    "openai",
	"ANTHROPIC_API_KEY": "anthropic",
	"GOOGLE_API_KEY":    "google",
	"MISTRAL_API_KEY":   "mistral",
	"GROQ_API_KEY":      "groq",
	"DEEPSEEK_API_KEY":  "deepseek",
	"XAI_API_KEY":       "xai",
	"COHERE_API_KEY":    "cohere",
}

// refreshProviderIfSecret checks if a secret name matches a provider key pattern
// and refreshes the JS-side provider cache if so.
// Uses KernelConfig.ProviderKeyMapping if set, otherwise defaultProviderKeyMapping.
func (k *Kernel) refreshProviderIfSecret(name, newValue string) {
	mapping := k.config.ProviderKeyMapping
	if mapping == nil {
		mapping = defaultProviderKeyMapping
	}

	provName, ok := mapping[name]
	if !ok {
		return
	}

	// Update the JS-side provider cache
	k.callJSSync("__brainkit.secrets.refreshProvider", map[string]string{
		"provider": provName,
		"apiKey":   newValue,
	})
}
