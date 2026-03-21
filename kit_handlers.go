package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// ---------------------------------------------------------------------------
// Bus Handler Registration
// ---------------------------------------------------------------------------

func (k *Kit) registerHandlers() {
	// If this Kit is part of a pool, use AsWorker for competing consumers
	var subOpts []bus.SubscribeOption
	if k.config.WorkerGroup != "" {
		subOpts = append(subOpts, bus.AsWorker(k.config.WorkerGroup))
	}

	// wrapHandler adapts the old Handler signature to the new On/ReplyFunc pattern.
	wrapHandler := func(h func(ctx context.Context, msg bus.Message) (*bus.Message, error)) func(bus.Message, bus.ReplyFunc) {
		return func(msg bus.Message, reply bus.ReplyFunc) {
			resp, err := h(context.Background(), msg)
			if err != nil {
				errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
				reply(errPayload)
				return
			}
			if resp != nil {
				reply(resp.Payload)
			}
		}
	}

	k.Bus.On("wasm.*", wrapHandler(k.wasm.HandleBusMessage), subOpts...)

	k.Bus.On("tools.*", wrapHandler(func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		switch msg.Topic {
		case "tools.resolve":
			return k.handleToolsResolve(ctx, msg)
		case "tools.call":
			return k.handleToolsCall(ctx, msg)
		case "tools.register":
			return k.handleToolsRegister(ctx, msg)
		case "tools.list":
			return k.handleToolsList(ctx, msg)
		default:
			return nil, fmt.Errorf("tools: unknown topic %q", msg.Topic)
		}
	}), subOpts...)

	k.Bus.On("mcp.*", wrapHandler(func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		if k.MCP == nil {
			return nil, fmt.Errorf("mcp: no MCP servers configured")
		}
		switch msg.Topic {
		case "mcp.listTools":
			tools := k.MCP.ListTools()
			data, _ := json.Marshal(tools)
			return &bus.Message{Payload: data}, nil
		case "mcp.callTool":
			var params struct {
				Server string          `json:"server"`
				Tool   string          `json:"tool"`
				Args   json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(msg.Payload, &params); err != nil {
				return nil, fmt.Errorf("mcp.callTool: %w", err)
			}
			result, err := k.MCP.CallTool(ctx, params.Server, params.Tool, params.Args)
			if err != nil {
				return nil, err
			}
			return &bus.Message{Payload: result}, nil
		default:
			return nil, fmt.Errorf("mcp: unknown topic %q", msg.Topic)
		}
	}), subOpts...)

	k.Bus.On("agents.*", wrapHandler(k.handleAgents), subOpts...)
	k.Bus.On("fs.*", wrapHandler(k.handleFs), subOpts...)
	k.Bus.On("ai.*", wrapHandler(k.handleAI), subOpts...)
	k.Bus.On("memory.*", wrapHandler(k.handleMemory), subOpts...)
	k.Bus.On("workflows.*", wrapHandler(k.handleWorkflows), subOpts...)
	k.Bus.On("vectors.*", wrapHandler(k.handleVectors), subOpts...)
	k.Bus.On("kit.*", wrapHandler(k.handleDeploy), subOpts...)

	// Plugin state handlers — plugins call GetState/SetState via typed messages
	pluginState := make(map[string]string)
	var pluginStateMu sync.Mutex
	k.Bus.On("plugin.state.*", wrapHandler(func(_ context.Context, msg bus.Message) (*bus.Message, error) {
		switch msg.Topic {
		case "plugin.state.get":
			var req struct {
				Key string `json:"key"`
			}
			json.Unmarshal(msg.Payload, &req)
			pluginStateMu.Lock()
			val := pluginState[req.Key]
			pluginStateMu.Unlock()
			result, _ := json.Marshal(map[string]string{"value": val})
			return &bus.Message{Payload: result}, nil
		case "plugin.state.set":
			var req struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			json.Unmarshal(msg.Payload, &req)
			pluginStateMu.Lock()
			pluginState[req.Key] = req.Value
			pluginStateMu.Unlock()
			result, _ := json.Marshal(map[string]bool{"ok": true})
			return &bus.Message{Payload: result}, nil
		default:
			return nil, fmt.Errorf("plugin.state: unknown topic %q", msg.Topic)
		}
	}), subOpts...)
}

// ---------------------------------------------------------------------------
// Tools Handlers
// ---------------------------------------------------------------------------

func (k *Kit) handleToolsCall(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.call: invalid request: %w", err)
	}

	tool, err := k.Tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	result, err := tool.Executor.Call(ctx, msg.CallerID, req.Input)
	if err != nil {
		return nil, err
	}

	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsRegister(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.register: invalid request: %w", err)
	}

	// Compose new-format name from caller identity.
	// CallerID is the owner context; req.Name is the short tool name.
	callerID := msg.CallerID
	var fullName string
	if registry.IsNewFormat(req.Name) {
		fullName = req.Name
	} else {
		fullName = registry.ComposeName(callerID, callerID, "0.0.0", req.Name)
	}

	k.Tools.Register(registry.RegisteredTool{
		Name:        fullName,
		ShortName:   req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
	})

	result, _ := json.Marshal(map[string]string{"registered": fullName})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsList(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Filter string `json:"filter"`
	}
	json.Unmarshal(msg.Payload, &req)

	toolList := k.Tools.List(req.Filter)
	var infos []map[string]any
	for _, t := range toolList {
		infos = append(infos, map[string]any{
			"name":        t.Name,
			"shortName":   t.ShortName,
			"description": t.Description,
		})
	}

	result, _ := json.Marshal(infos)
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsResolve(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.resolve: invalid request: %w", err)
	}

	tool, err := k.Tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	info := map[string]any{
		"name":        tool.Name,
		"shortName":   tool.ShortName,
		"description": tool.Description,
	}
	if tool.InputSchema != nil {
		info["inputSchema"] = string(tool.InputSchema)
	}

	result, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	return &bus.Message{Payload: result}, nil
}
