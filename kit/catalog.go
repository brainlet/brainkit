package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

type commandSpec struct {
	topic         string
	resultTopic   string
	validate      func(json.RawMessage) error
	encodeFailure func(error) (json.RawMessage, error)
	invokeKernel  func(context.Context, *Kernel, json.RawMessage) (json.RawMessage, error)
	invokeNode    func(context.Context, *Node, json.RawMessage) (json.RawMessage, error)
}

const legacyResultSuffix = "." + "resp"

func kernelCommand[Req, Resp messages.BrainkitMessage](handler func(context.Context, *Kernel, Req) (*Resp, error)) commandSpec {
	var req Req
	var resp Resp
	return commandSpec{
		topic:       req.BusTopic(),
		resultTopic: resp.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		encodeFailure: func(err error) (json.RawMessage, error) {
			return encodeCommandFailure[Resp](err)
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

func nodeCommand[Req, Resp messages.BrainkitMessage](handler func(context.Context, *Node, Req) (*Resp, error)) commandSpec {
	var req Req
	var resp Resp
	return commandSpec{
		topic:       req.BusTopic(),
		resultTopic: resp.BusTopic(),
		validate: func(payload json.RawMessage) error {
			_, err := decodeCommand[Req](payload, req.BusTopic())
			return err
		},
		encodeFailure: func(err error) (json.RawMessage, error) {
			return encodeCommandFailure[Resp](err)
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

func encodeCommandFailure[Resp any](err error) (json.RawMessage, error) {
	var out Resp
	carrier, ok := any(&out).(interface{ SetError(string) })
	if !ok {
		return nil, fmt.Errorf("command result %T does not embed messages.ResultMeta", out)
	}
	carrier.SetError(err.Error())
	return json.Marshal(out)
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
		bindings = append(bindings, messaging.RawCommandBinding{
			Name:          spec.topic,
			Topic:         spec.topic,
			ResultTopic:   spec.resultTopic,
			EncodeFailure: spec.encodeFailure,
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
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AiGenerateMsg) (*messages.AiGenerateResp, error) {
				return kernel.ai.Generate(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AiEmbedMsg) (*messages.AiEmbedResp, error) {
				return kernel.ai.Embed(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AiEmbedManyMsg) (*messages.AiEmbedManyResp, error) {
				return kernel.ai.EmbedMany(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AiGenerateObjectMsg) (*messages.AiGenerateObjectResp, error) {
				return kernel.ai.GenerateObject(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolCallMsg) (*messages.ToolCallResp, error) {
				return kernel.toolsDomain.Call(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolResolveMsg) (*messages.ToolResolveResp, error) {
				return kernel.toolsDomain.Resolve(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.ToolListMsg) (*messages.ToolListResp, error) {
				return kernel.toolsDomain.List(ctx, req)
			}),
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
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentRequestMsg) (*messages.AgentRequestResp, error) {
				return kernel.agentsDomain.Request(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AgentMessageMsg) (*messages.AgentMessageResp, error) {
				return kernel.agentsDomain.Message(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsReadMsg) (*messages.FsReadResp, error) {
				return kernel.fsDomain.Read(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsWriteMsg) (*messages.FsWriteResp, error) {
				return kernel.fsDomain.Write(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsListMsg) (*messages.FsListResp, error) {
				return kernel.fsDomain.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsStatMsg) (*messages.FsStatResp, error) {
				return kernel.fsDomain.Stat(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsDeleteMsg) (*messages.FsDeleteResp, error) {
				return kernel.fsDomain.Delete(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.FsMkdirMsg) (*messages.FsMkdirResp, error) {
				return kernel.fsDomain.Mkdir(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemoryCreateThreadMsg) (*messages.MemoryCreateThreadResp, error) {
				return kernel.memoryDomain.CreateThread(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemoryGetThreadMsg) (*messages.MemoryGetThreadResp, error) {
				return kernel.memoryDomain.GetThread(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemoryListThreadsMsg) (*messages.MemoryListThreadsResp, error) {
				return kernel.memoryDomain.ListThreads(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemorySaveMsg) (*messages.MemorySaveResp, error) {
				return kernel.memoryDomain.Save(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemoryRecallMsg) (*messages.MemoryRecallResp, error) {
				return kernel.memoryDomain.Recall(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.MemoryDeleteThreadMsg) (*messages.MemoryDeleteThreadResp, error) {
				return kernel.memoryDomain.DeleteThread(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WorkflowRunMsg) (*messages.WorkflowRunResp, error) {
				return kernel.workflowsDomain.Run(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WorkflowResumeMsg) (*messages.WorkflowResumeResp, error) {
				return kernel.workflowsDomain.Resume(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WorkflowCancelMsg) (*messages.WorkflowCancelResp, error) {
				return kernel.workflowsDomain.Cancel(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WorkflowStatusMsg) (*messages.WorkflowStatusResp, error) {
				return kernel.workflowsDomain.Status(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.VectorCreateIndexMsg) (*messages.VectorCreateIndexResp, error) {
				return kernel.vectorsDomain.CreateIndex(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.VectorDeleteIndexMsg) (*messages.VectorDeleteIndexResp, error) {
				return kernel.vectorsDomain.DeleteIndex(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.VectorListIndexesMsg) (*messages.VectorListIndexesResp, error) {
				return kernel.vectorsDomain.ListIndexes(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.VectorUpsertMsg) (*messages.VectorUpsertResp, error) {
				return kernel.vectorsDomain.Upsert(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.VectorQueryMsg) (*messages.VectorQueryResp, error) {
				return kernel.vectorsDomain.Query(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmCompileMsg) (*messages.WasmCompileResp, error) {
				return kernel.wasmDomainInst.Compile(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmRunMsg) (*messages.WasmRunResp, error) {
				return kernel.wasmDomainInst.Run(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmDeployMsg) (*messages.WasmDeployResp, error) {
				return kernel.wasmDomainInst.Deploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmUndeployMsg) (*messages.WasmUndeployResp, error) {
				return kernel.wasmDomainInst.Undeploy(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmListMsg) (*messages.WasmListResp, error) {
				return kernel.wasmDomainInst.List(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmGetMsg) (*messages.WasmGetResp, error) {
				return kernel.wasmDomainInst.Get(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmRemoveMsg) (*messages.WasmRemoveResp, error) {
				return kernel.wasmDomainInst.Remove(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.WasmDescribeMsg) (*messages.WasmDescribeResp, error) {
				return kernel.wasmDomainInst.Describe(ctx, req)
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
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.McpListToolsMsg) (*messages.McpListToolsResp, error) {
				return kernel.mcpDomainInst.ListTools(ctx, req)
			}),
			kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.McpCallToolMsg) (*messages.McpCallToolResp, error) {
				return kernel.mcpDomainInst.CallTool(ctx, req)
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
		}

		byTopic := make(map[string]commandSpec, len(specs))
		for _, spec := range specs {
			if strings.HasSuffix(spec.topic, legacyResultSuffix) || strings.HasSuffix(spec.topic, ".result") {
				panic(fmt.Sprintf("invalid command topic registered: %s", spec.topic))
			}
			if strings.HasSuffix(spec.resultTopic, legacyResultSuffix) || !strings.HasSuffix(spec.resultTopic, ".result") {
				panic(fmt.Sprintf("invalid command result topic registered: %s", spec.resultTopic))
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

// commandBindingsForKernel generates router bindings for a standalone Kernel.
// Kernel-only commands are bound; node-only commands (plugin.*) are skipped.
func commandBindingsForKernel(kernel *Kernel) []messaging.RawCommandBinding {
	catalog := commandCatalog()
	bindings := make([]messaging.RawCommandBinding, 0, len(catalog.ordered))
	for _, spec := range catalog.ordered {
		spec := spec
		if spec.invokeKernel == nil {
			continue // node-only command
		}
		bindings = append(bindings, messaging.RawCommandBinding{
			Name:          spec.topic,
			Topic:         spec.topic,
			ResultTopic:   spec.resultTopic,
			EncodeFailure: spec.encodeFailure,
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
