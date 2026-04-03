package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/tracing"
	"github.com/brainlet/brainkit/sdk/messages"
)

type commandSpec struct {
	topic        string
	validate     func(json.RawMessage) error
	invokeKernel func(context.Context, *Kernel, json.RawMessage) (json.RawMessage, error)
	invokeNode   func(context.Context, *Node, json.RawMessage) (json.RawMessage, error)
}


func kernelCommand[Req messages.BrainkitMessage, Resp any](handler func(context.Context, *Kernel, Req) (*Resp, error)) commandSpec {
	var req Req
	return commandSpec{
		topic: req.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		invokeKernel: func(ctx context.Context, kernel *Kernel, payload json.RawMessage) (json.RawMessage, error) {
			decoded, err := decodeCommand[Req](payload, req.BusTopic())
			if err != nil {
				return nil, err
			}
			out, err := handler(ctx, kernel, decoded)
			if err != nil {
				return nil, err
			}
			return json.Marshal(out)
		},
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			return kernelCommand(handler).invokeKernel(ctx, node.Kernel, payload)
		},
	}
}

func nodeCommand[Req messages.BrainkitMessage, Resp any](handler func(context.Context, *Node, Req) (*Resp, error)) commandSpec {
	var req Req
	return commandSpec{
		topic: req.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			decoded, err := decodeCommand[Req](payload, req.BusTopic())
			if err != nil {
				return nil, err
			}
			out, err := handler(ctx, node, decoded)
			if err != nil {
				return nil, err
			}
			return json.Marshal(out)
		},
	}
}

func decodeCommand[T any](payload json.RawMessage, topic string) (T, error) {
	var out T
	if len(payload) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, messaging.NewDecodeFailure(topic, err)
	}
	return out, nil
}



type commandRegistry struct {
	ordered []commandSpec
	byTopic map[string]commandSpec
}

func (r *commandRegistry) Lookup(topic string) (commandSpec, bool) {
	spec, ok := r.byTopic[topic]
	return spec, ok
}

func (r *commandRegistry) HasCommand(topic string) bool {
	_, ok := r.byTopic[topic]
	return ok
}

func (r *commandRegistry) Validate(topic string, payload json.RawMessage) error {
	spec, ok := r.byTopic[topic]
	if !ok || spec.validate == nil {
		return nil
	}
	return spec.validate(payload)
}

func (r *commandRegistry) BindingsForNode(node *Node) []messaging.RawCommandBinding {
	bindings := make([]messaging.RawCommandBinding, 0, len(r.ordered))
	for _, spec := range r.ordered {
		spec := spec
		if spec.invokeNode == nil && spec.invokeKernel == nil {
			continue
		}
		if shouldSkipCommand(spec.topic, node.Kernel) {
			continue
		}
		bindings = append(bindings, messaging.RawCommandBinding{
			Name:  spec.topic,
			Topic: spec.topic,
			Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
				if spec.invokeNode != nil {
					return spec.invokeNode(ctx, node, payload)
				}
				return spec.invokeKernel(ctx, node.Kernel, payload)
			},
		})
	}
	return bindings
}

var (
	commandCatalogOnce sync.Once
	commandCatalogInst *commandRegistry
)

func commandCatalog() *commandRegistry {
	commandCatalogOnce.Do(func() {
		specs := []commandSpec{
			// ── Tools ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolCallMsg) (*messages.ToolCallResp, error) {
				return kernel.toolsDomain.Call(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolResolveMsg) (*messages.ToolResolveResp, error) {
				return kernel.toolsDomain.Resolve(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolListMsg) (*messages.ToolListResp, error) {
				return kernel.toolsDomain.List(ctx, req)
			}),
			// ── Agents (registry ops only) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentListMsg) (*messages.AgentListResp, error) {
				filter := (*agentFilter)(nil)
				if req.Filter != nil {
					filter = &agentFilter{
						Capability: req.Filter.Capability,
						Model:      req.Filter.Model,
						Status:     req.Filter.Status,
					}
				}
				return kernel.agentsDomain.List(ctx, filter)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentDiscoverMsg) (*messages.AgentDiscoverResp, error) {
				return kernel.agentsDomain.Discover(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentGetStatusMsg) (*messages.AgentGetStatusResp, error) {
				return kernel.agentsDomain.GetStatus(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentSetStatusMsg) (*messages.AgentSetStatusResp, error) {
				return kernel.agentsDomain.SetStatus(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.KitDeployMsg) (*messages.KitDeployResp, error) {
				return kernel.lifecycle.Deploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.KitTeardownMsg) (*messages.KitTeardownResp, error) {
				return kernel.lifecycle.Teardown(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.KitRedeployMsg) (*messages.KitRedeployResp, error) {
				return kernel.lifecycle.Redeploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.KitListMsg) (*messages.KitListResp, error) {
				return kernel.lifecycle.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.KitDeployFileMsg) (*messages.KitDeployResp, error) {
				resources, err := DeployFile(ctx, kernel, req.Path)
				if err != nil {
					return nil, err
				}
				return &messages.KitDeployResp{
					Deployed:  true,
					Resources: resourceInfosToMessages(resources),
				}, nil
			}),
			// ── MCP (inlined) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.McpListToolsMsg) (*messages.McpListToolsResp, error) {
				if kernel.mcp == nil {
					return nil, ErrMCPNotConfigured
				}
				tools := kernel.mcp.ListTools()
				var infos []messages.McpToolInfo
				for _, t := range tools {
					infos = append(infos, messages.McpToolInfo{Name: t.Name, Server: t.ServerName, Description: t.Description})
				}
				return &messages.McpListToolsResp{Tools: infos}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.McpCallToolMsg) (*messages.McpCallToolResp, error) {
				if kernel.mcp == nil {
					return nil, ErrMCPNotConfigured
				}
				argsJSON, _ := json.Marshal(req.Args)
				result, err := kernel.mcp.CallTool(ctx, req.Server, req.Tool, argsJSON)
				if err != nil {
					return nil, err
				}
				return &messages.McpCallToolResp{Result: result}, nil
			}),
			// ── Registry (inlined) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RegistryHasMsg) (*messages.RegistryHasResp, error) {
				var found bool
				switch req.Category {
				case "provider":
					found = kernel.providers.HasAIProvider(req.Name)
				case "vectorStore":
					found = kernel.providers.HasVectorStore(req.Name)
				case "storage":
					found = kernel.providers.HasStorage(req.Name)
				}
				return &messages.RegistryHasResp{Found: found}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RegistryListMsg) (*messages.RegistryListResp, error) {
				var result any
				switch req.Category {
				case "provider":
					result = kernel.providers.ListAIProviders()
				case "vectorStore":
					result = kernel.providers.ListVectorStores()
				case "storage":
					result = kernel.providers.ListStorages()
				default:
					result = []any{}
				}
				b, _ := json.Marshal(result)
				return &messages.RegistryListResp{Items: b}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RegistryResolveMsg) (*messages.RegistryResolveResp, error) {
				var configJSON []byte
				switch req.Category {
				case "provider":
					if reg, ok := kernel.providers.GetAIProvider(req.Name); ok {
						configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
					}
				case "vectorStore":
					if reg, ok := kernel.providers.GetVectorStore(req.Name); ok {
						configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
					}
				case "storage":
					if reg, ok := kernel.providers.GetStorage(req.Name); ok {
						configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
					}
				}
				return &messages.RegistryResolveResp{Config: configJSON}, nil
			}),
			// ── Workflows (handlers in handlers_workflows.go) ──
			kernelCommand(handleWorkflowStart),
			kernelCommand(handleWorkflowStartAsync),
			kernelCommand(handleWorkflowStatus),
			kernelCommand(handleWorkflowResume),
			kernelCommand(handleWorkflowCancel),
			kernelCommand(handleWorkflowList),
			kernelCommand(handleWorkflowRuns),
			kernelCommand(handleWorkflowRestart),
			// ── Metrics ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MetricsGetMsg) (*messages.MetricsGetResp, error) {
				data, _ := json.Marshal(kernel.Metrics())
				return &messages.MetricsGetResp{Metrics: data}, nil
			}),
			// ── Tracing (inlined) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.TraceGetMsg) (*messages.TraceGetResp, error) {
				if kernel.config.TraceStore == nil {
					return &messages.TraceGetResp{Spans: json.RawMessage("[]")}, nil
				}
				spans, err := kernel.config.TraceStore.GetTrace(req.TraceID)
				if err != nil {
					return nil, err
				}
				data, _ := json.Marshal(spans)
				return &messages.TraceGetResp{Spans: data}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.TraceListMsg) (*messages.TraceListResp, error) {
				if kernel.config.TraceStore == nil {
					return &messages.TraceListResp{Traces: json.RawMessage("[]")}, nil
				}
				query := tracing.TraceQuery{Source: req.Source, Status: req.Status, Limit: req.Limit}
				if req.MinDuration > 0 {
					query.MinDuration = time.Duration(req.MinDuration) * time.Millisecond
				}
				traces, err := kernel.config.TraceStore.ListTraces(query)
				if err != nil {
					return nil, err
				}
				data, _ := json.Marshal(traces)
				return &messages.TraceListResp{Traces: data}, nil
			}),
			// ── RBAC Administration (inlined — no domain type needed) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RBACAssignMsg) (*messages.RBACAssignResp, error) {
				if kernel.rbac == nil {
					return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
				}
				if req.Source == "" {
					return nil, &sdkerrors.ValidationError{Field: "source", Message: "is required"}
				}
				if err := kernel.rbac.Assign(req.Source, req.Role); err != nil {
					return nil, err
				}
				return &messages.RBACAssignResp{Assigned: true}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RBACRevokeMsg) (*messages.RBACRevokeResp, error) {
				if kernel.rbac == nil {
					return nil, &sdkerrors.NotConfiguredError{Feature: "rbac"}
				}
				kernel.rbac.Revoke(req.Source)
				return &messages.RBACRevokeResp{Revoked: true}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RBACListMsg) (*messages.RBACListResp, error) {
				if kernel.rbac == nil {
					return &messages.RBACListResp{Assignments: []messages.RBACAssignmentInfo{}}, nil
				}
				assignments := kernel.rbac.ListAssignments()
				infos := make([]messages.RBACAssignmentInfo, len(assignments))
				for i, a := range assignments {
					infos[i] = messages.RBACAssignmentInfo{
						Source: a.Source, Role: a.Role,
						AssignedAt: a.AssignedAt.Format(time.RFC3339),
					}
				}
				return &messages.RBACListResp{Assignments: infos}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.RBACRolesMsg) (*messages.RBACRolesResp, error) {
				if kernel.rbac == nil {
					return &messages.RBACRolesResp{Roles: json.RawMessage("[]")}, nil
				}
				roles := kernel.rbac.ListRoles()
				data, _ := json.Marshal(roles)
				return &messages.RBACRolesResp{Roles: data}, nil
			}),
			// ── Secrets ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.SecretsSetMsg) (*messages.SecretsSetResp, error) {
				return kernel.secretsDomain.Set(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.SecretsGetMsg) (*messages.SecretsGetResp, error) {
				return kernel.secretsDomain.Get(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.SecretsDeleteMsg) (*messages.SecretsDeleteResp, error) {
				return kernel.secretsDomain.Delete(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.SecretsListMsg) (*messages.SecretsListResp, error) {
				return kernel.secretsDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.SecretsRotateMsg) (*messages.SecretsRotateResp, error) {
				return kernel.secretsDomain.Rotate(ctx, req)
			}),
			// ── Package Manager ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesSearchMsg) (*messages.PackagesSearchResp, error) {
				return kernel.packagesDomain.Search(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesInstallMsg) (*messages.PackagesInstallResp, error) {
				return kernel.packagesDomain.Install(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesRemoveMsg) (*messages.PackagesRemoveResp, error) {
				return kernel.packagesDomain.Remove(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesUpdateMsg) (*messages.PackagesUpdateResp, error) {
				return kernel.packagesDomain.Update(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesListMsg) (*messages.PackagesListResp, error) {
				return kernel.packagesDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackagesInfoMsg) (*messages.PackagesInfoResp, error) {
				return kernel.packagesDomain.Info(ctx, req)
			}),
			// ── Package Deployment ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackageDeployMsg) (*messages.PackageDeployResp, error) {
				return kernel.packageDeployDomain.Deploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackageTeardownMsg) (*messages.PackageTeardownResp, error) {
				return kernel.packageDeployDomain.Teardown(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackageRedeployMsg) (*messages.PackageRedeployResp, error) {
				return kernel.packageDeployDomain.Redeploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackageListDeployedMsg) (*messages.PackageListDeployedResp, error) {
				return kernel.packageDeployDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.PackageDeployInfoMsg) (*messages.PackageDeployInfoResp, error) {
				return kernel.packageDeployDomain.Info(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginManifestMsg) (*messages.PluginManifestResp, error) {
				return node.processPluginManifest(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginStateGetMsg) (*messages.PluginStateGetResp, error) {
				return node.getPluginState(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginStateSetMsg) (*messages.PluginStateSetResp, error) {
				return node.setPluginState(ctx, req)
			}),
			// ── Plugin Lifecycle ──
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginStartMsg) (*messages.PluginStartResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Start(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginStopMsg) (*messages.PluginStopResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Stop(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginRestartMsg) (*messages.PluginRestartResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Restart(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginListRunningMsg) (*messages.PluginListRunningResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.List(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PluginStatusMsg) (*messages.PluginStatusResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Status(ctx, req)
			}),
			// ── Testing ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.TestRunMsg) (*messages.TestRunResp, error) {
				return kernel.testingDomain.Run(ctx, req)
			}),
			// ── Peer Discovery ──
			nodeCommand(func(ctx context.Context, node *Node, req messages.PeersListMsg) (*messages.PeersListResp, error) {
				if node.discovery == nil {
					return &messages.PeersListResp{Peers: []messages.PeerInfo{}}, nil
				}
				peers, err := node.discovery.Browse()
				if err != nil {
					return nil, err
				}
				infos := make([]messages.PeerInfo, len(peers))
				for i, p := range peers {
					infos[i] = messages.PeerInfo{Name: p.Name, Namespace: p.Namespace, Address: p.Address, Meta: p.Meta}
				}
				return &messages.PeersListResp{Peers: infos}, nil
			}),
			nodeCommand(func(ctx context.Context, node *Node, req messages.PeersResolveMsg) (*messages.PeersResolveResp, error) {
				if node.discovery == nil {
					return nil, &sdkerrors.NotConfiguredError{Feature: "discovery"}
				}
				addr, err := node.discovery.Resolve(req.Name)
				if err != nil {
					return nil, err
				}
				return &messages.PeersResolveResp{Namespace: addr}, nil
			}),
		}

		byTopic := make(map[string]commandSpec, len(specs))
		for _, spec := range specs {
			if strings.HasSuffix(spec.topic, ".result") {
				panic(fmt.Sprintf("invalid command topic registered: %s", spec.topic))
			}
			if _, exists := byTopic[spec.topic]; exists {
				panic(fmt.Sprintf("duplicate command topic registered: %s", spec.topic))
			}
			byTopic[spec.topic] = spec
		}

		commandCatalogInst = &commandRegistry{
			ordered: specs,
			byTopic: byTopic,
		}
	})
	return commandCatalogInst
}

// shouldSkipCommand returns true if the command topic targets an unconfigured domain.
func shouldSkipCommand(topic string, kernel *Kernel) bool {
	if strings.HasPrefix(topic, "mcp.") && kernel.mcp == nil {
		return true
	}
	if strings.HasPrefix(topic, "rbac.") && kernel.rbac == nil {
		return true
	}
	if strings.HasPrefix(topic, "trace.") && kernel.config.TraceStore == nil {
		return true
	}
	return false
}

// commandBindingsForKernel generates router bindings for a standalone Kernel.
// Kernel-only commands are bound; node-only and unconfigured-domain commands are skipped.
func commandBindingsForKernel(kernel *Kernel) []messaging.RawCommandBinding {
	catalog := commandCatalog()
	bindings := make([]messaging.RawCommandBinding, 0, len(catalog.ordered))
	for _, spec := range catalog.ordered {
		spec := spec
		if spec.invokeKernel == nil {
			continue // node-only command
		}
		if shouldSkipCommand(spec.topic, kernel) {
			continue // domain not configured
		}
		bindings = append(bindings, messaging.RawCommandBinding{
			Name:  spec.topic,
			Topic: spec.topic,
			Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
				return spec.invokeKernel(ctx, kernel, payload)
			},
		})
	}
	return bindings
}

// commandBindingsForNode generates router bindings for a Node.
// Includes both kernel commands (delegated to node.Kernel) and node-specific commands.
func commandBindingsForNode(node *Node) []messaging.RawCommandBinding {
	return commandCatalog().BindingsForNode(node)
}
