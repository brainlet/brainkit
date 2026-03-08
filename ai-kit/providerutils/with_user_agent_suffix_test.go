// Ported from: packages/provider-utils/src/with-user-agent-suffix.test.ts
package providerutils

import "testing"

func TestWithUserAgentSuffix_NewHeader(t *testing.T) {
	headers := map[string]string{
		"content-type":  "application/json",
		"authorization": "Bearer token123",
	}
	result := WithUserAgentSuffix(headers, "ai-sdk/0.0.0-test", "provider/test-openai")
	if result["user-agent"] != "ai-sdk/0.0.0-test provider/test-openai" {
		t.Errorf("unexpected user-agent: %q", result["user-agent"])
	}
	if result["content-type"] != "application/json" {
		t.Errorf("unexpected content-type: %q", result["content-type"])
	}
}

func TestWithUserAgentSuffix_AppendToExisting(t *testing.T) {
	headers := map[string]string{
		"user-agent": "TestApp/0.0.0-test",
		"accept":     "application/json",
	}
	result := WithUserAgentSuffix(headers, "ai-sdk/0.0.0-test", "provider/test-anthropic")
	expected := "TestApp/0.0.0-test ai-sdk/0.0.0-test provider/test-anthropic"
	if result["user-agent"] != expected {
		t.Errorf("expected %q, got %q", expected, result["user-agent"])
	}
}
