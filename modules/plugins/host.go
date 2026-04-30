package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/tools"
	coretracing "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/plugin"
)

type pluginHost interface {
	PublishRaw(context.Context, string, json.RawMessage) (string, error)
	TransportKind() string
	Store() Store
	Logger() *slog.Logger
	ReportError(error, types.ErrorContext)
	SetPluginChecker(bkmodule.PluginChecker)
	SetPluginRestarter(plugincap.Restarter)
	Audit() *auditpkg.Recorder
	Tools() *tools.ToolRegistry
	Tracer() *coretracing.Tracer
	Remote() *transport.RemoteClient
	Namespace() string
	CallerID() string
	SecretStore() types.SecretStore
	ShutdownSignal() <-chan struct{}
}

type pluginMountExt struct {
	transportKind      string
	secretStore        types.SecretStore
	shutdownSignal     <-chan struct{}
	remote             *transport.RemoteClient
	tools              *tools.ToolRegistry
	tracer             *coretracing.Tracer
	audit              *auditpkg.Recorder
	reportError        func(error, types.ErrorContext)
	setPluginChecker   func(bkmodule.PluginChecker)
	setPluginRestarter func(plugincap.Restarter)
	namespace          string
	callerID           string
}

type mountedPluginHost struct {
	host bkmodule.Host
	ext  pluginMountExt
}

func newMountedPluginHost(host bkmodule.Host) (pluginHost, error) {
	transportKind, err := bkmodule.RequireCapability[string](host, bkmodule.CapabilityTransportKind)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	shutdownSignal, err := bkmodule.RequireCapability[<-chan struct{}](host, bkmodule.CapabilityShutdownSignal)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	remote, err := bkmodule.RequireCapability[*transport.RemoteClient](host, bkmodule.CapabilityRemoteClient)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	toolRegistry, err := bkmodule.RequireCapability[*tools.ToolRegistry](host, bkmodule.CapabilityToolRegistry)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	tracer, err := bkmodule.RequireCapability[*coretracing.Tracer](host, bkmodule.CapabilityTracer)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	audit, err := bkmodule.RequireCapability[*auditpkg.Recorder](host, bkmodule.CapabilityAuditRecorder)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	reportError, err := bkmodule.RequireCapability[func(error, types.ErrorContext)](host, bkmodule.CapabilityReportError)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	setPluginChecker, err := bkmodule.RequireCapability[func(bkmodule.PluginChecker)](host, bkmodule.CapabilitySetPluginChecker)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	setPluginRestarter, err := bkmodule.RequireCapability[func(plugincap.Restarter)](host, bkmodule.CapabilitySetPluginRestarter)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	namespace, err := bkmodule.RequireCapability[string](host, bkmodule.CapabilityNamespace)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	callerID, err := bkmodule.RequireCapability[string](host, bkmodule.CapabilityCallerID)
	if err != nil {
		return nil, fmt.Errorf("plugins: %w", err)
	}
	secretStore, _ := bkmodule.Capability[types.SecretStore](host, bkmodule.CapabilitySecretStore)

	return mountedPluginHost{host: host, ext: pluginMountExt{
		transportKind:      transportKind,
		secretStore:        secretStore,
		shutdownSignal:     shutdownSignal,
		remote:             remote,
		tools:              toolRegistry,
		tracer:             tracer,
		audit:              audit,
		reportError:        reportError,
		setPluginChecker:   setPluginChecker,
		setPluginRestarter: setPluginRestarter,
		namespace:          namespace,
		callerID:           callerID,
	}}, nil
}

func (h mountedPluginHost) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return h.host.Runtime().PublishRaw(ctx, topic, payload)
}
func (h mountedPluginHost) TransportKind() string { return h.ext.transportKind }
func (h mountedPluginHost) Store() Store {
	store, _ := h.host.Store().(Store)
	return store
}
func (h mountedPluginHost) Logger() *slog.Logger { return h.host.Logger() }
func (h mountedPluginHost) ReportError(err error, ctx types.ErrorContext) {
	h.ext.reportError(err, ctx)
}
func (h mountedPluginHost) SetPluginChecker(checker bkmodule.PluginChecker) {
	h.ext.setPluginChecker(checker)
}
func (h mountedPluginHost) SetPluginRestarter(restarter plugincap.Restarter) {
	h.ext.setPluginRestarter(restarter)
}
func (h mountedPluginHost) Audit() *auditpkg.Recorder       { return h.ext.audit }
func (h mountedPluginHost) Tools() *tools.ToolRegistry      { return h.ext.tools }
func (h mountedPluginHost) Tracer() *coretracing.Tracer     { return h.ext.tracer }
func (h mountedPluginHost) Remote() *transport.RemoteClient { return h.ext.remote }
func (h mountedPluginHost) Namespace() string               { return h.ext.namespace }
func (h mountedPluginHost) CallerID() string                { return h.ext.callerID }
func (h mountedPluginHost) SecretStore() types.SecretStore  { return h.ext.secretStore }
func (h mountedPluginHost) ShutdownSignal() <-chan struct{} { return h.ext.shutdownSignal }
