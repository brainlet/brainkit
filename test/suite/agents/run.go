package agents

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("agents", func(t *testing.T) {
		t.Run("list_empty", func(t *testing.T) { testListEmpty(t, env) })
		t.Run("discover_no_match", func(t *testing.T) { testDiscoverNoMatch(t, env) })
		t.Run("get_status_not_found", func(t *testing.T) { testGetStatusNotFound(t, env) })
		t.Run("set_status_not_found", func(t *testing.T) { testSetStatusNotFound(t, env) })
		t.Run("set_status_invalid", func(t *testing.T) { testSetStatusInvalid(t, env) })

		// ai.go — AI agent tests (require OPENAI_API_KEY)
		t.Run("deploy_agent_then_list", func(t *testing.T) { testDeployAgentThenList(t, env) })

		// surface.go — surface AI tests (require OPENAI_API_KEY)
		t.Run("surface_generate_text_real", func(t *testing.T) { testSurfaceGenerateTextReal(t, env) })
		t.Run("surface_agent_generate", func(t *testing.T) { testSurfaceAgentGenerate(t, env) })
		t.Run("surface_agent_with_tool", func(t *testing.T) { testSurfaceAgentWithTool(t, env) })
		t.Run("surface_bus_service_ai_proxy", func(t *testing.T) { testSurfaceBusServiceAIProxy(t, env) })

		// hitl.go — generateWithApproval bus-based tool approval (require OPENAI_API_KEY)
		t.Run("hitl_generate_with_approval_no_mastra_wrap", func(t *testing.T) { testGenerateWithApprovalNoMastraWrap(t, env) })
		t.Run("hitl_generate_with_approval_mastra_wrap_approve", func(t *testing.T) { testGenerateWithApprovalWithMastraWrap_Approve(t, env) })
		t.Run("hitl_generate_with_approval_mastra_wrap_decline", func(t *testing.T) { testGenerateWithApprovalWithMastraWrap_Decline(t, env) })
		t.Run("hitl_generate_with_approval_decline_with_retries", func(t *testing.T) { testGenerateWithApprovalDeclineWithRetries(t, env) })

		// guardrails.go — input processor guardrails (require OPENAI_API_KEY)
		t.Run("guardrails_detection_agent_direct", func(t *testing.T) { testGuardrailsDetectionAgentDirect(t, env) })
		t.Run("guardrails_prompt_injection_rewrite", func(t *testing.T) { testGuardrailsPromptInjectionRewrite(t, env) })
	})
}
