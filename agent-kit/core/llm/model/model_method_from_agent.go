// Ported from: packages/core/src/llm/model/model-method-from-agent.ts
package model

import (
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// ---------------------------------------------------------------------------
// AgentMethodType
// ---------------------------------------------------------------------------

// AgentMethodType represents agent method types.
// TS: import type { AgentMethodType } from '../../agent';
// STUB REASON: Cannot import agent.AgentMethodType due to circular dependency:
// agent imports llm/model. Both define identical string enum values.
type AgentMethodType string

const (
	AgentMethodGenerate       AgentMethodType = "generate"
	AgentMethodGenerateLegacy AgentMethodType = "generateLegacy"
	AgentMethodStream         AgentMethodType = "stream"
	AgentMethodStreamLegacy   AgentMethodType = "streamLegacy"
)

// ---------------------------------------------------------------------------
// GetModelMethodFromAgentMethod
// ---------------------------------------------------------------------------

// GetModelMethodFromAgentMethod converts an agent method type to a model method type.
//
// TS:
//
//	export function getModelMethodFromAgentMethod(methodType: AgentMethodType): ModelMethodType {
//	  if (methodType === 'generate' || methodType === 'generateLegacy') { return 'generate'; }
//	  else if (methodType === 'stream' || methodType === 'streamLegacy') { return 'stream'; }
//	  else { throw new MastraError({ id: 'INVALID_METHOD_TYPE', ... }); }
//	}
func GetModelMethodFromAgentMethod(methodType AgentMethodType) (ModelMethodType, error) {
	switch methodType {
	case AgentMethodGenerate, AgentMethodGenerateLegacy:
		return ModelMethodGenerate, nil
	case AgentMethodStream, AgentMethodStreamLegacy:
		return ModelMethodStream, nil
	default:
		return "", mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "INVALID_METHOD_TYPE",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
}
