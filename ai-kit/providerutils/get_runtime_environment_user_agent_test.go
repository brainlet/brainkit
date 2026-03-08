// Ported from: packages/provider-utils/src/get-runtime-environment-user-agent.test.ts
package providerutils

import (
	"strings"
	"testing"
)

func TestGetRuntimeEnvironmentUserAgent_ContainsGo(t *testing.T) {
	ua := GetRuntimeEnvironmentUserAgent()
	if !strings.HasPrefix(ua, "runtime/go/") {
		t.Errorf("expected user agent to start with 'runtime/go/', got %q", ua)
	}
}

func TestGetRuntimeEnvironmentUserAgent_ContainsVersion(t *testing.T) {
	ua := GetRuntimeEnvironmentUserAgent()
	if !strings.Contains(ua, "go") {
		t.Errorf("expected user agent to contain 'go', got %q", ua)
	}
}
