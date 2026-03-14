package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiembed "github.com/brainlet/brainkit/ai-embed"
	"github.com/brainlet/brainkit/bus"
)

// AIService wraps ai-embed for direct LLM calls through the bus.
type AIService struct {
	kit    *Kit
	client *aiembed.Client
}

// newAIService creates an AIService with its own ai-embed Client.
// The ai-embed Client has its own QuickJS runtime — separate from any sandbox.
func newAIService(kit *Kit) (*AIService, error) {
	// Build env vars for AI SDK provider resolution
	envVars := make(map[string]string)
	for name, pc := range kit.config.Providers {
		// AI SDK resolves providers via process.env
		envKey := strings.ToUpper(name) + "_API_KEY"
		envVars[envKey] = pc.APIKey
	}
	for k, v := range kit.config.EnvVars {
		envVars[k] = v
	}

	client, err := aiembed.NewClient(aiembed.ClientConfig{
		EnvVars: envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create AI service: %w", err)
	}

	return &AIService{kit: kit, client: client}, nil
}

func (s *AIService) close() {
	if s.client != nil {
		s.client.Close()
	}
}

// handleBusMessage routes ai.* bus messages to the appropriate ai-embed method.
func (s *AIService) handleBusMessage(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "ai.generate":
		return s.handleGenerate(ctx, msg)
	default:
		return nil, fmt.Errorf("ai service: unknown topic %q", msg.Topic)
	}
}

// aiGenerateRequest is the JSON payload for ai.generate from .ts code.
type aiGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt,omitempty"`
	System string `json:"system,omitempty"`
}

func (s *AIService) handleGenerate(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req aiGenerateRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("ai.generate: invalid request: %w", err)
	}

	// Build ai-embed Model — resolve "provider/model" format
	model := aiembed.ModelFromString(req.Model)

	// If we have provider config, attach it
	provider, modelID := splitModelString(req.Model)
	if provider != "" {
		if pc, ok := s.kit.config.Providers[provider]; ok {
			model = aiembed.Model{
				ID: modelID,
				Provider: &aiembed.ProviderConfig{
					Provider: provider,
					APIKey:   pc.APIKey,
					BaseURL:  pc.BaseURL,
				},
			}
		}
	}

	result, err := s.client.GenerateText(aiembed.GenerateTextParams{
		Model:  model,
		Prompt: req.Prompt,
		System: req.System,
	})
	if err != nil {
		return nil, fmt.Errorf("ai.generate: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("ai.generate: marshal result: %w", err)
	}

	return &bus.Message{Payload: resultJSON}, nil
}

func splitModelString(model string) (provider, modelID string) {
	idx := strings.IndexByte(model, '/')
	if idx < 0 {
		return "", model
	}
	return model[:idx], model[idx+1:]
}
