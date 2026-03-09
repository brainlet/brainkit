// Ported from: packages/core/src/llm/model/resolve-model.ts
package model

import (
	"fmt"
)

// ModelConfigFunc is a function that dynamically resolves a MastraModelConfig.
// TS: ({ requestContext, mastra }) => MastraModelConfig | Promise<MastraModelConfig>
type ModelConfigFunc func(args ModelConfigFuncArgs) (MastraModelConfig, error)

// ModelConfigFuncArgs holds the arguments passed to a ModelConfigFunc.
type ModelConfigFuncArgs struct {
	// RequestContext holds the request-scoped context.
	RequestContext RequestContext
	// Mastra is a stub for the Mastra instance.
	Mastra MastraRef
}

// IsOpenAICompatibleObjectConfig checks if a model config is an OpenAICompatibleConfig.
//
// Returns true if the config is an OpenAICompatibleConfig struct (not a language model
// instance). This is the Go equivalent of the TS type guard.
//
// TS logic:
//
//	if (typeof modelConfig === 'object' && 'specificationVersion' in modelConfig) return false;
//	if (typeof modelConfig === 'object' && !('model' in modelConfig)) {
//	  if ('id' in modelConfig) return true;
//	  if ('providerId' in modelConfig && 'modelId' in modelConfig) return true;
//	}
//	return false;
func IsOpenAICompatibleObjectConfig(config any) bool {
	// If it's a language model (has specificationVersion), it's not an OpenAICompatibleConfig
	if _, ok := config.(MastraLanguageModel); ok {
		return false
	}
	switch c := config.(type) {
	case OpenAICompatibleConfig:
		return c.HasID() || c.HasProviderModel()
	case *OpenAICompatibleConfig:
		return c != nil && (c.HasID() || c.HasProviderModel())
	default:
		return false
	}
}

// DoStreamDoGenerate is an interface for checking if a model has doStream/doGenerate methods.
// Used for unknown specificationVersion models from third-party providers.
type DoStreamDoGenerate interface {
	DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
	DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
}

// ListGatewaysFunc is an interface for Mastra instances that can list gateways.
// STUB REASON: Cannot import core.Mastra due to circular dependency: core imports llm/model.
// This minimal interface captures only the ListGateways method needed by ResolveModelConfig.
type ListGatewaysFunc interface {
	ListGateways() map[string]MastraModelGateway
}

// ResolveModelConfig resolves a model configuration to a language model instance.
//
// Supports:
//   - Magic strings like "openai/gpt-4o"
//   - Config objects like { ID: "openai/gpt-4o", APIKey: "..." }
//   - Direct LanguageModel instances
//   - Dynamic functions that return any of the above
//
// TS signature:
//
//	export async function resolveModelConfig(
//	  modelConfig: MastraModelConfig | ((...) => MastraModelConfig | Promise<MastraModelConfig>),
//	  requestContext: RequestContext = new RequestContext(),
//	  mastra?: Mastra,
//	): Promise<MastraLanguageModel | MastraLegacyLanguageModel>
//
// Returns either a MastraLanguageModel (modern V2/V3) or a MastraLegacyLanguageModel (V1).
// The variadic requestContext accepts optional [requestContext, mastra] args.
func ResolveModelConfig(modelConfig any, customGateways []MastraModelGateway, requestContext ...any) (any, error) {
	// Extract optional args
	var reqCtx RequestContext
	var mastra MastraRef
	if len(requestContext) > 0 {
		if rc, ok := requestContext[0].(RequestContext); ok {
			reqCtx = rc
		}
	}
	if len(requestContext) > 1 {
		if m, ok := requestContext[1].(MastraRef); ok {
			mastra = m
		}
	}

	// If it's a function, resolve it first
	// TS: if (typeof modelConfig === 'function') { modelConfig = await modelConfig({ requestContext, mastra }); }
	if fn, ok := modelConfig.(ModelConfigFunc); ok {
		resolved, err := fn(ModelConfigFuncArgs{
			RequestContext: reqCtx,
			Mastra:         mastra,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dynamic model config: %w", err)
		}
		modelConfig = resolved
	}

	// Filter out custom language model instances (already wrapped)
	// TS: if (modelConfig instanceof ModelRouterLanguageModel || modelConfig instanceof AISDKV5LanguageModel || modelConfig instanceof AISDKV6LanguageModel)
	if m, ok := modelConfig.(*ModelRouterLanguageModel); ok {
		return m, nil
	}
	if m, ok := modelConfig.(*AISDKV5LanguageModel); ok {
		return m, nil
	}
	if m, ok := modelConfig.(*AISDKV6LanguageModelStub); ok {
		return m, nil
	}

	// Check for already-resolved language model instances
	// TS: if (typeof modelConfig === 'object' && 'specificationVersion' in modelConfig)
	if m, ok := modelConfig.(MastraLanguageModel); ok {
		specVersion := m.SpecificationVersion()
		switch specVersion {
		case "v2":
			// TS: return new AISDKV5LanguageModel(modelConfig as LanguageModelV2);
			if v2, ok := modelConfig.(LanguageModelV2); ok {
				return NewAISDKV5LanguageModel(v2), nil
			}
			return m, nil
		case "v3":
			// TS: return new AISDKV6LanguageModel(modelConfig as LanguageModelV3);
			if v3, ok := modelConfig.(LanguageModelV3); ok {
				return NewAISDKV6LanguageModelStub(v3), nil
			}
			return m, nil
		case "v1":
			// TS: return modelConfig;
			return m, nil
		default:
			// Unknown specificationVersion from a third-party provider (e.g. ollama-ai-provider-v2).
			// If the model has doStream/doGenerate methods, wrap it as a modern model
			// to prevent the stream()/streamLegacy() catch-22 where neither method accepts the model.
			// TS: if (typeof (modelConfig as any).doStream === 'function' && typeof (modelConfig as any).doGenerate === 'function') {
			//       return new AISDKV5LanguageModel(modelConfig as LanguageModelV2);
			//     }
			if v2, ok := modelConfig.(LanguageModelV2); ok {
				return NewAISDKV5LanguageModel(v2), nil
			}
			return m, nil
		}
	}

	// Get custom gateways from mastra if available and no custom gateways provided
	// TS: const gatewayRecord = mastra?.listGateways();
	//     const customGateways = gatewayRecord ? Object.values(gatewayRecord) : undefined;
	if customGateways == nil && mastra != nil {
		if lg, ok := mastra.(ListGatewaysFunc); ok {
			gatewayRecord := lg.ListGateways()
			if gatewayRecord != nil {
				for _, gw := range gatewayRecord {
					customGateways = append(customGateways, gw)
				}
			}
		}
	}

	// Check for string (magic string like "openai/gpt-4o")
	// TS: if (typeof modelConfig === 'string' || isOpenAICompatibleObjectConfig(modelConfig))
	if s, ok := modelConfig.(string); ok {
		return NewModelRouterLanguageModel(s, customGateways)
	}

	// Check for OpenAICompatibleConfig
	if IsOpenAICompatibleObjectConfig(modelConfig) {
		switch c := modelConfig.(type) {
		case OpenAICompatibleConfig:
			return NewModelRouterLanguageModel(c, customGateways)
		case *OpenAICompatibleConfig:
			return NewModelRouterLanguageModel(*c, customGateways)
		}
	}

	return nil, fmt.Errorf("invalid model configuration provided")
}
