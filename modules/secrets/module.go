// Package secrets owns secrets.* bus commands as a hot-mountable Kit module.
package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	coresecrets "github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Module exposes secrets.set/get/delete/list/rotate. Construct via New and
// include in brainkit.Config.Modules when the runtime should expose secret
// management over the bus.
type Module struct {
	store                 coresecrets.SecretStore
	bus                   busPublisher
	audit                 *auditpkg.Recorder
	callerID              string
	pluginRestarter       func() any
	refreshProviderSecret func(string, string)
}

type busPublisher interface {
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error)
}

type pluginRestarter interface {
	ListRunningPlugins() []types.RunningPlugin
	RestartPlugin(ctx context.Context, name string) error
}

// New creates the secrets module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "secrets" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers secrets.* command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	store, err := bkmodule.RequireCapability[coresecrets.SecretStore](host, bkmodule.CapabilitySecretStore)
	if err != nil {
		return fmt.Errorf("secrets: %w", err)
	}
	m.store = store
	m.bus = host.Messages()
	m.audit, _ = bkmodule.Capability[*auditpkg.Recorder](host, bkmodule.CapabilityAuditRecorder)
	m.callerID, _ = bkmodule.Capability[string](host, bkmodule.CapabilityCallerID)
	m.pluginRestarter, _ = bkmodule.Capability[func() any](host, bkmodule.CapabilityPluginRestarter)
	m.refreshProviderSecret, _ = bkmodule.Capability[func(string, string)](host, bkmodule.CapabilityRefreshProviderSecret)

	host.Scope().Defer(func(context.Context) error {
		m.store = nil
		m.bus = nil
		m.audit = nil
		m.callerID = ""
		m.pluginRestarter = nil
		m.refreshProviderSecret = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.Set),
		bkmodule.Command(m.Get),
		bkmodule.Command(m.Delete),
		bkmodule.Command(m.List),
		bkmodule.Command(m.Rotate),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module from Kit capabilities. Command handles are owned by
// the module scope, so unmounting unregisters them.
func (m *Module) Close() error {
	m.store = nil
	m.bus = nil
	m.audit = nil
	m.callerID = ""
	m.pluginRestarter = nil
	m.refreshProviderSecret = nil
	return nil
}

// Factory is the registered ModuleFactory for secrets.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the secrets module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "secrets",
		Status:  bkmodule.StatusStable,
		Summary: "Secret management bus commands (secrets.set, get, delete, list, rotate).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[secretmsg.SecretsDeleteMsg, secretmsg.SecretsDeleteResp](),
			bkmodule.CommandMessage[secretmsg.SecretsGetMsg, secretmsg.SecretsGetResp](),
			bkmodule.CommandMessage[secretmsg.SecretsListMsg, secretmsg.SecretsListResp](),
			bkmodule.CommandMessage[secretmsg.SecretsRotateMsg, secretmsg.SecretsRotateResp](),
			bkmodule.CommandMessage[secretmsg.SecretsSetMsg, secretmsg.SecretsSetResp](),
		},
		Events: []bkmodule.MessageDescriptor{
			bkmodule.EventMessage[secretmsg.SecretsAccessedEvent](),
			bkmodule.EventMessage[secretmsg.SecretsDeletedEvent](),
			bkmodule.EventMessage[secretmsg.SecretsRotatedEvent](),
			bkmodule.EventMessage[secretmsg.SecretsStoredEvent](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[coresecrets.SecretStore](bkmodule.CapabilitySecretStore),
		},
	}
}

func init() { bkmodule.Register("secrets", Factory{}) }

// Set handles secrets.set.
func (m *Module) Set(ctx context.Context, req secretmsg.SecretsSetMsg) (*secretmsg.SecretsSetResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.store.Set(ctx, req.Name, req.Value); err != nil {
		return nil, err
	}

	version := m.version(ctx, req.Name)
	m.emit(ctx, secretmsg.SecretsStoredEvent{Name: req.Name, Version: version, Timestamp: time.Now().Format(time.RFC3339)})
	m.audit.SecretSet(req.Name, m.callerID)

	return &secretmsg.SecretsSetResp{Stored: true, Version: version}, nil
}

// Get handles secrets.get.
func (m *Module) Get(ctx context.Context, req secretmsg.SecretsGetMsg) (*secretmsg.SecretsGetResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	val, err := m.store.Get(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	m.emit(ctx, secretmsg.SecretsAccessedEvent{Name: req.Name, Accessor: m.callerID, Timestamp: time.Now().Format(time.RFC3339)})
	return &secretmsg.SecretsGetResp{Value: val}, nil
}

// Delete handles secrets.delete.
func (m *Module) Delete(ctx context.Context, req secretmsg.SecretsDeleteMsg) (*secretmsg.SecretsDeleteResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.store.Delete(ctx, req.Name); err != nil {
		return nil, err
	}
	m.emit(ctx, secretmsg.SecretsDeletedEvent{Name: req.Name, Timestamp: time.Now().Format(time.RFC3339)})
	m.audit.SecretDeleted(req.Name, m.callerID)
	return &secretmsg.SecretsDeleteResp{Deleted: true}, nil
}

// List handles secrets.list.
func (m *Module) List(ctx context.Context, _ secretmsg.SecretsListMsg) (*secretmsg.SecretsListResp, error) {
	metas, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]secretmsg.SecretMetaInfo, 0, len(metas))
	for _, meta := range metas {
		infos = append(infos, secretmsg.SecretMetaInfo{
			Name:      meta.Name,
			CreatedAt: meta.CreatedAt.Format(time.RFC3339),
			UpdatedAt: meta.UpdatedAt.Format(time.RFC3339),
			Version:   meta.Version,
		})
	}
	return &secretmsg.SecretsListResp{Secrets: infos}, nil
}

// Rotate handles secrets.rotate.
func (m *Module) Rotate(ctx context.Context, req secretmsg.SecretsRotateMsg) (*secretmsg.SecretsRotateResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.store.Set(ctx, req.Name, req.NewValue); err != nil {
		return nil, err
	}

	version := m.version(ctx, req.Name)
	restartedPlugins := m.restartPlugins(ctx, req)

	if m.refreshProviderSecret != nil {
		m.refreshProviderSecret(req.Name, req.NewValue)
	}

	m.emit(ctx, secretmsg.SecretsRotatedEvent{
		Name:             req.Name,
		Version:          version,
		RestartedPlugins: restartedPlugins,
		Timestamp:        time.Now().Format(time.RFC3339),
	})
	m.audit.SecretRotated(req.Name, m.callerID)

	return &secretmsg.SecretsRotateResp{
		Rotated:          true,
		Version:          version,
		RestartedPlugins: restartedPlugins,
	}, nil
}

func (m *Module) emit(ctx context.Context, event sdk.BrainkitMessage) {
	if m.bus == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = m.bus.PublishRaw(ctx, event.BusTopic(), payload)
}

func (m *Module) version(ctx context.Context, name string) int {
	version := 1
	metas, _ := m.store.List(ctx)
	for _, meta := range metas {
		if meta.Name == name {
			version = meta.Version
			break
		}
	}
	return version
}

func (m *Module) restartPlugins(ctx context.Context, req secretmsg.SecretsRotateMsg) []string {
	if !req.Restart || m.pluginRestarter == nil {
		return nil
	}
	restarter, ok := m.pluginRestarter().(pluginRestarter)
	if !ok || restarter == nil {
		return nil
	}
	var restarted []string
	for _, plugin := range restarter.ListRunningPlugins() {
		if pluginUsesSecret(plugin, req.Name) {
			if err := restarter.RestartPlugin(ctx, plugin.Name); err == nil {
				restarted = append(restarted, plugin.Name)
			}
		}
	}
	return restarted
}

func pluginUsesSecret(plugin types.RunningPlugin, secretName string) bool {
	for _, value := range plugin.Config.Env {
		if value == "$secret:"+secretName {
			return true
		}
	}
	return false
}
