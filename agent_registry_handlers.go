package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
)

// handleAgents is the bus handler for all agents.* topics.
func (k *Kit) handleAgents(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "agents.register":
		return k.handleAgentRegister(ctx, msg)
	case "agents.unregister":
		return k.handleAgentUnregister(ctx, msg)
	case "agents.list":
		return k.handleAgentList(ctx, msg)
	case "agents.discover":
		return k.handleAgentDiscover(ctx, msg)
	case "agents.get-status":
		return k.handleAgentGetStatus(ctx, msg)
	case "agents.set-status":
		return k.handleAgentSetStatus(ctx, msg)
	case "agents.request":
		return k.handleAgentRequest(ctx, msg)
	case "agents.message":
		return k.handleAgentMessage(ctx, msg)
	default:
		return nil, fmt.Errorf("agents: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleAgentRegister(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
		Model        string   `json:"model"`
		Kit          string   `json:"kit"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.register: invalid request: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("agents.register: name is required")
	}

	k.agentReg.register(AgentInfo{
		Name:         req.Name,
		Capabilities: req.Capabilities,
		Model:        req.Model,
		Status:       "idle",
		Kit:          req.Kit,
	})

	result, _ := json.Marshal(map[string]string{"registered": req.Name})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentUnregister(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.unregister: invalid request: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("agents.unregister: name is required")
	}

	found := k.agentReg.unregister(req.Name)
	if !found {
		return nil, fmt.Errorf("agents.unregister: agent %q not found", req.Name)
	}

	result, _ := json.Marshal(map[string]bool{"ok": true})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentList(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Filter *agentFilter `json:"filter,omitempty"`
	}
	json.Unmarshal(msg.Payload, &req)

	agents := k.agentReg.list(req.Filter)
	if agents == nil {
		agents = []AgentInfo{}
	}

	result, _ := json.Marshal(agents)
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentDiscover(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var filter agentFilter
	if err := json.Unmarshal(msg.Payload, &filter); err != nil {
		return nil, fmt.Errorf("agents.discover: invalid request: %w", err)
	}

	agents := k.agentReg.list(&filter)
	if agents == nil {
		agents = []AgentInfo{}
	}

	result, _ := json.Marshal(agents)
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentGetStatus(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.get-status: invalid request: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("agents.get-status: name is required")
	}

	info := k.agentReg.get(req.Name)
	if info == nil {
		return nil, fmt.Errorf("agents.get-status: agent %q not found", req.Name)
	}

	result, _ := json.Marshal(map[string]string{
		"name":   info.Name,
		"status": info.Status,
	})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentSetStatus(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.set-status: invalid request: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("agents.set-status: name is required")
	}
	if req.Status == "" {
		return nil, fmt.Errorf("agents.set-status: status is required")
	}

	switch req.Status {
	case "idle", "busy", "error":
	default:
		return nil, fmt.Errorf("agents.set-status: invalid status %q (must be idle|busy|error)", req.Status)
	}

	found := k.agentReg.setStatus(req.Name, req.Status)
	if !found {
		return nil, fmt.Errorf("agents.set-status: agent %q not found", req.Name)
	}

	result, _ := json.Marshal(map[string]bool{"ok": true})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleAgentRequest(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name   string `json:"name"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.request: invalid request: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("agents.request: name is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("agents.request: prompt is required")
	}

	info := k.agentReg.get(req.Name)
	if info == nil {
		return nil, fmt.Errorf("agents.request: agent %q not found", req.Name)
	}

	code := fmt.Sprintf(`
		var _agent = globalThis.__kit_agent_registry[%q];
		if (!_agent) throw new Error("agent not found in JS registry: " + %q);
		var _result = await _agent.generate(%q);
		return JSON.stringify({ text: _result.text || "" });
	`, req.Name, req.Name, req.Prompt)

	resultJSON, err := k.EvalTS(ctx, "__agents_request.ts", code)
	if err != nil {
		return nil, fmt.Errorf("agents.request: generate failed: %w", err)
	}

	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleAgentMessage(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Target  string          `json:"target"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("agents.message: invalid request: %w", err)
	}
	if req.Target == "" {
		return nil, fmt.Errorf("agents.message: target is required")
	}

	info := k.agentReg.get(req.Target)
	if info == nil {
		return nil, fmt.Errorf("agents.message: agent %q not found", req.Target)
	}

	agentTopic := "agent." + req.Target + ".message"
	if err := k.Bus.Send(bus.Message{
		Topic:    agentTopic,
		CallerID: msg.CallerID,
		Payload:  req.Payload,
	}); err != nil {
		return nil, fmt.Errorf("agents.message: delivery failed: %w", err)
	}

	result, _ := json.Marshal(map[string]bool{"delivered": true})
	return &bus.Message{Payload: result}, nil
}
