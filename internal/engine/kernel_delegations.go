package engine

import (
	"context"
	"encoding/json"
	"fmt"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	provreg "github.com/brainlet/brainkit/internal/providers"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/types"
	"log/slog"
)

// --- sdk.Runtime implementation ---

// Namespace returns the runtime namespace.
func (k *Kernel) Namespace() string { return k.namespace }

// CallerID returns the runtime identity.
func (k *Kernel) CallerID() string { return k.callerID }

// Logger returns the structured logger.
func (k *Kernel) Logger() *slog.Logger { return k.logger }

// CreateAgent creates a persistent agent in the runtime.
func (k *Kernel) CreateAgent(cfg agentembed.AgentConfig) (*agentembed.Agent, error) {
	return k.agents.CreateAgent(cfg)
}

// --- Deployment delegation ---

// ListResources returns all tracked resources, optionally filtered by type.
func (k *Kernel) ListResources(resourceType ...string) ([]types.ResourceInfo, error) {
	return k.deploymentMgr.ListResources(resourceType...)
}

// ResourcesFrom returns all resources created by a specific .ts file.
func (k *Kernel) ResourcesFrom(filename string) ([]types.ResourceInfo, error) {
	return k.deploymentMgr.ResourcesFrom(filename)
}

// TeardownFile removes all resources created by a specific .ts file.
func (k *Kernel) TeardownFile(filename string) (int, error) {
	return k.deploymentMgr.TeardownFile(filename)
}

// RemoveResource removes a specific resource by type and ID.
func (k *Kernel) RemoveResource(resourceType, id string) error {
	return k.deploymentMgr.RemoveResource(resourceType, id)
}

// --- Eval delegation ---

// evalDomain marshals a request into JS globals and evaluates code atomically.
// Replaces per-domain evalAI/evalMemory/evalVector/evalWorkflow methods.
func (k *Kernel) evalDomain(ctx context.Context, req any, filename, code string) (json.RawMessage, error) {
	reqJSON, _ := json.Marshal(req)
	wrappedCode := fmt.Sprintf(`
		globalThis.__pending_req = %s;
		%s
	`, string(reqJSON), code)
	resultJSON, err := k.EvalTS(ctx, filename, wrappedCode)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resultJSON), nil
}

// EvalTS runs .ts-style code with brainkit infrastructure imports destructured.
func (k *Kernel) EvalTS(ctx context.Context, filename, code string) (string, error) {
	return k.deploymentMgr.EvalTS(ctx, filename, code)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kernel) EvalModule(ctx context.Context, filename, code string) (string, error) {
	return k.deploymentMgr.EvalModule(ctx, filename, code)
}

// RegisterTool is a convenience method for registering typed Go tools.
func RegisterTool[T any](k *Kernel, name string, tool toolreg.TypedTool[T]) error {
	return toolreg.Register(k.Tools, name, tool)
}

// --- Provider Registry delegation ---

// RegisterAIProvider registers a typed AI provider at runtime.
// Injects env vars into the JS runtime's process.env.
func (k *Kernel) RegisterAIProvider(name string, typ provreg.AIProviderType, config any) error {
	reg := provreg.AIProviderRegistration{Type: typ, Config: config}
	return k.providers.RegisterAIProvider(name, reg)
}

// UnregisterAIProvider removes an AI provider.
func (k *Kernel) UnregisterAIProvider(name string) { k.providers.UnregisterAIProvider(name) }

// ListAIProviders returns all registered AI providers.
func (k *Kernel) ListAIProviders() []provreg.ProviderInfo { return k.providers.ListAIProviders() }

// RegisterVectorStore registers a typed vector store at runtime.
func (k *Kernel) RegisterVectorStore(name string, typ provreg.VectorStoreType, config any) error {
	return k.providers.RegisterVectorStore(name, provreg.VectorStoreRegistration{Type: typ, Config: config})
}

// UnregisterVectorStore removes a vector store.
func (k *Kernel) UnregisterVectorStore(name string) { k.providers.UnregisterVectorStore(name) }

// ListVectorStores returns all registered vector stores.
func (k *Kernel) ListVectorStores() []provreg.VectorStoreInfo { return k.providers.ListVectorStores() }

// RegisterStorage registers a typed Mastra storage at runtime.
func (k *Kernel) RegisterStorage(name string, typ provreg.StorageType, config any) error {
	return k.providers.RegisterStorage(name, provreg.StorageRegistration{Type: typ, Config: config})
}

// UnregisterStorage removes a Mastra storage.
func (k *Kernel) UnregisterStorage(name string) { k.providers.UnregisterStorage(name) }

// ListStorages returns all registered Mastra storages.
func (k *Kernel) ListStorages() []provreg.StorageInfo { return k.providers.ListStorages() }

// currentDeploymentSource returns the deployment source currently executing on the JS thread.
// Used for tracing span attribution and audit source tracking.
func (k *Kernel) currentDeploymentSource() string {
	return k.deploymentMgr.currentSource
}

func (k *Kernel) setCurrentSource(source string) {
	k.deploymentMgr.setCurrentSource(source)
}
