package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/bus"
)

func (k *Kit) handleDeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "kit.deploy":
		return k.handleKitDeploy(ctx, msg)
	case "kit.teardown":
		return k.handleKitTeardown(ctx, msg)
	case "kit.list":
		return k.handleKitList(ctx, msg)
	case "kit.redeploy":
		return k.handleKitRedeploy(ctx, msg)
	default:
		return nil, fmt.Errorf("kit: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleKitDeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Source string `json:"source"`
		Code   string `json:"code"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("kit.deploy: %w", err)
	}
	if req.Source == "" {
		return nil, fmt.Errorf("kit.deploy: source is required")
	}
	if req.Code == "" {
		return nil, fmt.Errorf("kit.deploy: code is required")
	}

	resources, err := k.Deploy(ctx, req.Source, req.Code)
	if err != nil {
		return nil, err
	}
	result, _ := json.Marshal(map[string]any{
		"deployed": true, "resources": resources,
	})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleKitTeardown(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("kit.teardown: %w", err)
	}
	if req.Source == "" {
		return nil, fmt.Errorf("kit.teardown: source is required")
	}

	removed, err := k.Teardown(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	result, _ := json.Marshal(map[string]int{"removed": removed})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleKitList(_ context.Context, _ bus.Message) (*bus.Message, error) {
	deployments := k.ListDeployments()
	result, _ := json.Marshal(deployments)
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleKitRedeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Source string `json:"source"`
		Code   string `json:"code"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("kit.redeploy: %w", err)
	}

	resources, err := k.Redeploy(ctx, req.Source, req.Code)
	if err != nil {
		return nil, err
	}
	result, _ := json.Marshal(map[string]any{
		"deployed": true, "resources": resources,
	})
	return &bus.Message{Payload: result}, nil
}
