package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

type commandSpec struct {
	topic        string
	validate     func(json.RawMessage) error
	invokeKernel func(context.Context, *Kernel, json.RawMessage) (json.RawMessage, error)
	invokeNode   func(context.Context, *Node, json.RawMessage) (json.RawMessage, error)
}

// CommandSpec is the opaque registration handle produced by MakeCommand.
// Modules build one via brainkit.Command and register it through Kit.RegisterCommand.
type CommandSpec = commandSpec

// MakeCommand builds a CommandSpec from a handler that only sees context + Req.
// Used by the public brainkit.Command generic wrapper — handlers capture any Kit
// / Module state they need through closures, so they don't need a *Kernel arg.
func MakeCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, Req) (*Resp, error)) CommandSpec {
	var zero Req
	topic := zero.BusTopic()
	invoke := func(ctx context.Context, kernel *Kernel, payload json.RawMessage) (json.RawMessage, error) {
		decoded, err := decodeCommand[Req](payload, topic)
		if err != nil {
			return nil, err
		}
		cmdStart := time.Now()
		out, err := handler(ctx, decoded)
		cmdDuration := time.Since(cmdStart)
		callerID := transport.CallerIDFromContext(ctx)
		kernel.audit.BusCommandCompleted(topic, callerID, cmdDuration)
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, nil
		}
		return json.Marshal(out)
	}
	return commandSpec{
		topic: topic,
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, topic)
			return err
		},
		invokeKernel: invoke,
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			return invoke(ctx, node.Kernel, payload)
		},
	}
}


func kernelCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, *Kernel, Req) (*Resp, error)) commandSpec {
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
			cmdStart := time.Now()
			out, err := handler(ctx, kernel, decoded)
			cmdDuration := time.Since(cmdStart)
			callerID := transport.CallerIDFromContext(ctx)
			if err != nil {
				kernel.audit.BusCommandCompleted(req.BusTopic(), callerID, cmdDuration)
				return nil, err
			}
			kernel.audit.BusCommandCompleted(req.BusTopic(), callerID, cmdDuration)
			if out == nil {
				return nil, nil
			}
			return json.Marshal(out)
		},
		invokeNode: func(ctx context.Context, node *Node, payload json.RawMessage) (json.RawMessage, error) {
			return kernelCommand(handler).invokeKernel(ctx, node.Kernel, payload)
		},
	}
}

func nodeCommand[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, *Node, Req) (*Resp, error)) commandSpec {
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
		return out, transport.NewDecodeFailure(topic, err)
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

func (r *commandRegistry) BindingsForNode(node *Node) []transport.RawCommandBinding {
	bindings := make([]transport.RawCommandBinding, 0, len(r.ordered))
	for _, spec := range r.ordered {
		spec := spec
		if spec.invokeNode == nil && spec.invokeKernel == nil {
			continue
		}
		bindings = append(bindings, transport.RawCommandBinding{
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

func buildCommandCatalog() *commandRegistry {
	specs := []commandSpec{
			// ── Tools ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ToolCallMsg) (*sdk.ToolCallResp, error) {
				return kernel.toolsDomain.Call(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ToolResolveMsg) (*sdk.ToolResolveResp, error) {
				return kernel.toolsDomain.Resolve(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ToolListMsg) (*sdk.ToolListResp, error) {
				return kernel.toolsDomain.List(ctx, req)
			}),
			// ── Agents (registry ops only) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AgentListMsg) (*sdk.AgentListResp, error) {
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
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AgentDiscoverMsg) (*sdk.AgentDiscoverResp, error) {
				return kernel.agentsDomain.Discover(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AgentGetStatusMsg) (*sdk.AgentGetStatusResp, error) {
				return kernel.agentsDomain.GetStatus(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AgentSetStatusMsg) (*sdk.AgentSetStatusResp, error) {
				return kernel.agentsDomain.SetStatus(ctx, req)
			}),
			// ── SetDraining ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.KitSetDrainingMsg) (*sdk.KitSetDrainingResp, error) {
				kernel.SetDraining(req.Draining)
				return &sdk.KitSetDrainingResp{Draining: req.Draining}, nil
			}),
			// ── Eval (unified; dispatch on Mode: script | ts | module) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.KitEvalMsg) (*sdk.KitEvalResp, error) {
				mode := req.Mode
				if mode == "" {
					// Infer from source extension; empty source means script.
					if strings.HasSuffix(req.Source, ".ts") {
						mode = "ts"
					} else {
						mode = "script"
					}
				}
				switch mode {
				case "ts":
					source := req.Source
					if source == "" {
						source = "__eval_ts.ts"
					}
					result, err := kernel.EvalTS(ctx, source, req.Code)
					if err != nil {
						return nil, err
					}
					return &sdk.KitEvalResp{Result: result}, nil
				case "module":
					source := req.Source
					if source == "" {
						source = "__eval_module.ts"
					}
					result, err := kernel.EvalModule(ctx, source, req.Code)
					if err != nil {
						return nil, err
					}
					return &sdk.KitEvalResp{Result: result}, nil
				case "script":
					source := "__cli_eval_" + uuid.NewString() + ".ts"
					if _, err := kernel.Deploy(ctx, source, req.Code); err != nil {
						return nil, err
					}
					defer kernel.Teardown(ctx, source)
					result, _ := kernel.EvalTS(ctx, "__read_eval.ts", `return globalThis.__module_result || "null";`)
					return &sdk.KitEvalResp{Result: result}, nil
				default:
					return nil, &sdkerrors.ValidationError{Field: "mode", Message: "unknown eval mode: " + mode + " (want script|ts|module)"}
				}
			}),
			// ── Send (Go-side request-reply — no JS thread involvement) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.KitSendMsg) (*sdk.KitSendResp, error) {
				correlationID := uuid.NewString()
				replyTo := req.Topic + ".reply." + correlationID

				replyCh := make(chan sdk.Message, 1)
				unsub, err := kernel.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
					select {
					case replyCh <- msg:
					default:
					}
				})
				if err != nil {
					return nil, err
				}
				defer unsub()

				pubCtx := transport.WithPublishMeta(ctx, correlationID, replyTo)
				if _, err := kernel.PublishRaw(pubCtx, req.Topic, req.Payload); err != nil {
					return nil, err
				}

				select {
				case msg := <-replyCh:
					return &sdk.KitSendResp{Payload: msg.Payload}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}),
			// ── Cluster identity ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ClusterPeersMsg) (*sdk.ClusterPeersResp, error) {
				return &sdk.ClusterPeersResp{
					Peers: []sdk.ClusterPeerInfo{{
						ClusterID: kernel.config.ClusterID,
						RuntimeID: kernel.config.RuntimeID,
						Namespace: kernel.config.Namespace,
						CallerID:  kernel.config.CallerID,
						StartedAt: kernel.startedAt.Format("2006-01-02T15:04:05Z07:00"),
					}},
				}, nil
			}),
			// ── Health (bus) ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.KitHealthMsg) (*sdk.KitHealthResp, error) {
				data, _ := json.Marshal(kernel.Health(ctx))
				return &sdk.KitHealthResp{Health: data}, nil
			}),
			// ── Registry ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.RegistryHasMsg) (*sdk.RegistryHasResp, error) {
				return kernel.registryDomain.Has(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.RegistryListMsg) (*sdk.RegistryListResp, error) {
				return kernel.registryDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.RegistryResolveMsg) (*sdk.RegistryResolveResp, error) {
				return kernel.registryDomain.Resolve(ctx, req)
			}),
			// ── Workflows (moved to modules/workflow) ──
			// ── Metrics ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.MetricsGetMsg) (*sdk.MetricsGetResp, error) {
				return kernel.metricsDomain.Get(ctx, req)
			}),
			// ── Tracing (moved to modules/tracing) ──
			// ── Audit ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AuditQueryMsg) (*sdk.AuditQueryResp, error) {
				return newAuditDomain(auditStoreFromKernel(kernel)).Query(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AuditStatsMsg) (*sdk.AuditStatsResp, error) {
				return newAuditDomain(auditStoreFromKernel(kernel)).Stats(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.AuditPruneMsg) (*sdk.AuditPruneResp, error) {
				return newAuditDomain(auditStoreFromKernel(kernel)).Prune(ctx, req)
			}),
			// ── Secrets ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.SecretsSetMsg) (*sdk.SecretsSetResp, error) {
				return kernel.secretsDomain.Set(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.SecretsGetMsg) (*sdk.SecretsGetResp, error) {
				return kernel.secretsDomain.Get(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.SecretsDeleteMsg) (*sdk.SecretsDeleteResp, error) {
				return kernel.secretsDomain.Delete(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.SecretsListMsg) (*sdk.SecretsListResp, error) {
				return kernel.secretsDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.SecretsRotateMsg) (*sdk.SecretsRotateResp, error) {
				return kernel.secretsDomain.Rotate(ctx, req)
			}),
			// ── Package Deployment ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.PackageDeployMsg) (*sdk.PackageDeployResp, error) {
				return kernel.packageDeployDomain.Deploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.PackageTeardownMsg) (*sdk.PackageTeardownResp, error) {
				return kernel.packageDeployDomain.Teardown(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.PackageListDeployedMsg) (*sdk.PackageListDeployedResp, error) {
				return kernel.packageDeployDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.PackageDeployInfoMsg) (*sdk.PackageDeployInfoResp, error) {
				return kernel.packageDeployDomain.Info(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginManifestMsg) (*sdk.PluginManifestResp, error) {
				return node.processPluginManifest(ctx, req)
			}),
			// ── Plugin Lifecycle ──
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginStartMsg) (*sdk.PluginStartResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Start(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginStopMsg) (*sdk.PluginStopResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Stop(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginRestartMsg) (*sdk.PluginRestartResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Restart(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginListRunningMsg) (*sdk.PluginListRunningResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.List(ctx, req)
			}),
			nodeCommand(func(ctx context.Context, node *Node, req sdk.PluginStatusMsg) (*sdk.PluginStatusResp, error) {
				if node.pluginLifecycle == nil {
					node.pluginLifecycle = newPluginLifecycleDomain(node)
				}
				return node.pluginLifecycle.Status(ctx, req)
			}),
			// ── Testing ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.TestRunMsg) (*sdk.TestRunResp, error) {
				return kernel.testingDomain.Run(ctx, req)
			}),
			// ── Provider Management ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ProviderAddMsg) (*sdk.ProviderAddResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				config, err := deserializeProviderConfig(req.Type, req.Config)
				if err != nil {
					return nil, err
				}
				if err := kernel.RegisterAIProvider(req.Name, provreg.AIProviderType(req.Type), config); err != nil {
					return nil, err
				}
				return &sdk.ProviderAddResp{Added: true}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.ProviderRemoveMsg) (*sdk.ProviderRemoveResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				kernel.UnregisterAIProvider(req.Name)
				return &sdk.ProviderRemoveResp{Removed: true}, nil
			}),
			// ── Storage Management ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.StorageAddMsg) (*sdk.StorageAddResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				cfg, err := deserializeStorageConfig(req.Type, req.Config)
				if err != nil {
					return nil, err
				}
				if err := kernel.AddStorage(req.Name, cfg); err != nil {
					return nil, err
				}
				return &sdk.StorageAddResp{Added: true}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.StorageRemoveMsg) (*sdk.StorageRemoveResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				if err := kernel.RemoveStorage(req.Name); err != nil {
					return nil, err
				}
				return &sdk.StorageRemoveResp{Removed: true}, nil
			}),
			// ── Vector Store Management ──
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.VectorAddMsg) (*sdk.VectorAddResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				if err := kernel.RegisterVectorStore(req.Name, provreg.VectorStoreType(req.Type), nil); err != nil {
					return nil, err
				}
				return &sdk.VectorAddResp{Added: true}, nil
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.VectorRemoveMsg) (*sdk.VectorRemoveResp, error) {
				if req.Name == "" {
					return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
				}
				kernel.UnregisterVectorStore(req.Name)
				return &sdk.VectorRemoveResp{Removed: true}, nil
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

	return &commandRegistry{
		ordered: specs,
		byTopic: byTopic,
	}
}

// commandBindingsForKernel generates router bindings for a standalone Kernel.
// Kernel-only commands are bound; node-only and unconfigured-domain commands are skipped.
func commandBindingsForKernel(kernel *Kernel) []transport.RawCommandBinding {
	bindings := make([]transport.RawCommandBinding, 0, len(kernel.catalog.ordered))
	for _, spec := range kernel.catalog.ordered {
		spec := spec
		if spec.invokeKernel == nil {
			continue // node-only command
		}
		bindings = append(bindings, transport.RawCommandBinding{
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
func commandBindingsForNode(node *Node) []transport.RawCommandBinding {
	return node.Kernel.catalog.BindingsForNode(node)
}

// RegisterCommand adds a command to the per-instance catalog.
// Called by modules during Init to register their bus commands.
// Panics on duplicate topic (same as core catalog construction).
func (k *Kernel) RegisterCommand(spec commandSpec) {
	if _, exists := k.catalog.byTopic[spec.topic]; exists {
		panic(fmt.Sprintf("duplicate command topic registered: %s", spec.topic))
	}
	k.catalog.byTopic[spec.topic] = spec
	k.catalog.ordered = append(k.catalog.ordered, spec)
}
