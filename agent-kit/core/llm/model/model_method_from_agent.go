// Ported from: packages/core/src/llm/model/model-method-from-agent.ts
package model

import (
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/interfaces"
)

// ---------------------------------------------------------------------------
// AgentMethodType
// ---------------------------------------------------------------------------

// AgentMethodType is the shared agent method type, defined in core/interfaces
// to break the circular dependency between agent and llm/model packages.
type AgentMethodType = interfaces.AgentMethodType

// Re-export constants for backward compatibility. Consumers of model.AgentMethod*
// continue to work unchanged.
const (
	AgentMethodGenerate       = interfaces.AgentMethodGenerate
	AgentMethodGenerateLegacy = interfaces.AgentMethodGenerateLegacy
	AgentMethodStream         = interfaces.AgentMethodStream
	AgentMethodStreamLegacy   = interfaces.AgentMethodStreamLegacy
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
