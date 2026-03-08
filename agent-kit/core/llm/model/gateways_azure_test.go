// Ported from: packages/core/src/llm/model/gateways/azure.test.ts
package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model/gateways"
)

// ---------------------------------------------------------------------------
// HTTP mocking infrastructure for Azure tests
// ---------------------------------------------------------------------------

// mockRoundTripper intercepts HTTP requests and returns mock responses.
type mockRoundTripper struct {
	mu       sync.Mutex
	calls    []mockHTTPCall
	handlers []mockHTTPHandler
}

type mockHTTPCall struct {
	URL     string
	Method  string
	Headers http.Header
	Body    string
}

type mockHTTPHandler struct {
	matchFn    func(req *http.Request) bool
	responseFn func(req *http.Request) *http.Response
}

func newMockRoundTripper() *mockRoundTripper {
	return &mockRoundTripper{}
}

func (m *mockRoundTripper) addHandler(matchFn func(req *http.Request) bool, responseFn func(req *http.Request) *http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, mockHTTPHandler{matchFn: matchFn, responseFn: responseFn})
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockHTTPCall{
		URL:     req.URL.String(),
		Method:  req.Method,
		Headers: req.Header.Clone(),
	})

	for _, h := range m.handlers {
		if h.matchFn(req) {
			return h.responseFn(req), nil
		}
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       http.NoBody,
	}, nil
}

func (m *mockRoundTripper) getCalls() []mockHTTPCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockHTTPCall, len(m.calls))
	copy(result, m.calls)
	return result
}

func (m *mockRoundTripper) clearCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

// jsonResponseWithBody returns an http.Response with a JSON body.
func jsonResponseWithBody(statusCode int, body any) *http.Response {
	data, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io_NopCloser(strings.NewReader(string(data))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

// textResponseWithBody returns an http.Response with a text body.
func textResponseWithBody(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io_NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
	}
}

// io_NopCloser wraps a reader with a no-op Close method.
func io_NopCloser(r *strings.Reader) readCloser {
	return readCloser{r}
}

type readCloser struct {
	*strings.Reader
}

func (readCloser) Close() error { return nil }

// withMockHTTP replaces http.DefaultTransport for the duration of the test.
func withMockHTTP(t *testing.T) *mockRoundTripper {
	t.Helper()
	mock := newMockRoundTripper()
	origTransport := http.DefaultTransport
	http.DefaultTransport = mock
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})
	return mock
}

// ---------------------------------------------------------------------------
// Tests: Configuration Validation
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_ConfigurationValidation(t *testing.T) {
	t.Run("should return error if resourceName missing", func(t *testing.T) {
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			APIKey:      "test-key",
			Deployments: []string{"gpt-4"},
		})
		if err == nil {
			t.Fatal("expected error for missing resourceName")
		}
		if !strings.Contains(err.Error(), "resourceName is required") {
			t.Errorf("expected error to contain 'resourceName is required', got: %s", err.Error())
		}
	})

	t.Run("should return error if apiKey missing", func(t *testing.T) {
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			Deployments:  []string{"gpt-4"},
		})
		if err == nil {
			t.Fatal("expected error for missing apiKey")
		}
		if !strings.Contains(err.Error(), "apiKey is required") {
			t.Errorf("expected error to contain 'apiKey is required', got: %s", err.Error())
		}
	})

	t.Run("should allow neither deployments nor management", func(t *testing.T) {
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("should not error if both deployments and management provided", func(t *testing.T) {
		// In Go the constructor doesn't warn to console, it just proceeds.
		// The TS test checks that console.warn is called; in Go we just verify no error.
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{"gpt-4"},
			Management: &gateways.AzureManagementConfig{
				TenantID:       "tenant",
				ClientID:       "client",
				ClientSecret:   "secret",
				SubscriptionID: "sub",
				ResourceGroup:  "rg",
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("should return error if management credentials incomplete", func(t *testing.T) {
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID: "tenant",
				ClientID: "client",
			},
		})
		if err == nil {
			t.Fatal("expected error for incomplete management credentials")
		}
		if !strings.Contains(err.Error(), "management credentials incomplete") &&
			!strings.Contains(err.Error(), "Management credentials incomplete") {
			t.Errorf("expected error about management credentials, got: %s", err.Error())
		}
	})

	t.Run("should validate all missing management fields", func(t *testing.T) {
		_, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management:   &gateways.AzureManagementConfig{},
		})
		if err == nil {
			t.Fatal("expected error for empty management config")
		}
		errMsg := err.Error()
		for _, field := range []string{"tenantId", "clientId", "clientSecret", "subscriptionId", "resourceGroup"} {
			if !strings.Contains(errMsg, field) {
				t.Errorf("expected error to mention '%s', got: %s", field, errMsg)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Static Deployments Mode
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_StaticDeploymentsMode(t *testing.T) {
	t.Run("should return static deployments without API calls", func(t *testing.T) {
		mock := withMockHTTP(t)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{"gpt-4-prod", "gpt-35-turbo-dev"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error fetching providers: %v", err)
		}

		azureConfig, ok := providers["azure-openai"]
		if !ok {
			t.Fatal("expected 'azure-openai' in providers")
		}

		if len(azureConfig.Models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(azureConfig.Models))
		}
		if azureConfig.Models[0] != "gpt-4-prod" || azureConfig.Models[1] != "gpt-35-turbo-dev" {
			t.Errorf("unexpected models: %v", azureConfig.Models)
		}
		if azureConfig.Name != "Azure OpenAI" {
			t.Errorf("expected name 'Azure OpenAI', got '%s'", azureConfig.Name)
		}
		if azureConfig.Gateway != "azure-openai" {
			t.Errorf("expected gateway 'azure-openai', got '%s'", azureConfig.Gateway)
		}

		calls := mock.getCalls()
		if len(calls) != 0 {
			t.Errorf("expected no HTTP calls, got %d", len(calls))
		}
	})

	t.Run("should use static deployments even if management provided", func(t *testing.T) {
		mock := withMockHTTP(t)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{"gpt-4"},
			Management: &gateways.AzureManagementConfig{
				TenantID:       "tenant",
				ClientID:       "client",
				ClientSecret:   "secret",
				SubscriptionID: "sub",
				ResourceGroup:  "rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 1 || azureConfig.Models[0] != "gpt-4" {
			t.Errorf("expected ['gpt-4'], got %v", azureConfig.Models)
		}

		calls := mock.getCalls()
		if len(calls) != 0 {
			t.Errorf("expected no HTTP calls, got %d", len(calls))
		}
	})

	t.Run("should return empty models for empty deployments without management", func(t *testing.T) {
		mock := withMockHTTP(t)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 0 {
			t.Errorf("expected empty models, got %v", azureConfig.Models)
		}

		calls := mock.getCalls()
		if len(calls) != 0 {
			t.Errorf("expected no HTTP calls, got %d", len(calls))
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: No Configuration Mode
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_NoConfigurationMode(t *testing.T) {
	t.Run("should return empty models when neither deployments nor management provided", func(t *testing.T) {
		mock := withMockHTTP(t)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 0 {
			t.Errorf("expected empty models, got %v", azureConfig.Models)
		}
		if azureConfig.Name != "Azure OpenAI" {
			t.Errorf("expected name 'Azure OpenAI', got '%s'", azureConfig.Name)
		}

		calls := mock.getCalls()
		if len(calls) != 0 {
			t.Errorf("expected no HTTP calls, got %d", len(calls))
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Discovery Mode
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_DiscoveryMode(t *testing.T) {
	mockTokenResponse := map[string]any{
		"token_type":   "Bearer",
		"expires_in":   float64(3600),
		"access_token": "mock-access-token",
	}

	mockDeploymentsResponse := map[string]any{
		"value": []map[string]any{
			{
				"name": "my-gpt4",
				"properties": map[string]any{
					"model":             map[string]any{"name": "gpt-4", "version": "0613", "format": "OpenAI"},
					"provisioningState": "Succeeded",
				},
			},
			{
				"name": "staging-gpt-4o",
				"properties": map[string]any{
					"model":             map[string]any{"name": "gpt-4o", "version": "2024-05-13", "format": "OpenAI"},
					"provisioningState": "Succeeded",
				},
			},
			{
				"name": "creating-deployment",
				"properties": map[string]any{
					"model":             map[string]any{"name": "gpt-35-turbo", "version": "0613", "format": "OpenAI"},
					"provisioningState": "Creating",
				},
			},
		},
	}

	t.Run("should fetch token and deployments from Management API", func(t *testing.T) {
		mock := withMockHTTP(t)

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockTokenResponse)
			},
		)
		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "management.azure.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockDeploymentsResponse)
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := mock.getCalls()

		// Verify token call
		var tokenCall *mockHTTPCall
		for i := range calls {
			if strings.Contains(calls[i].URL, "login.microsoftonline.com") {
				tokenCall = &calls[i]
				break
			}
		}
		if tokenCall == nil {
			t.Fatal("expected a call to login.microsoftonline.com")
		}
		if !strings.Contains(tokenCall.URL, "test-tenant") {
			t.Errorf("expected token URL to contain 'test-tenant', got: %s", tokenCall.URL)
		}
		if tokenCall.Method != "POST" {
			t.Errorf("expected POST method for token call, got: %s", tokenCall.Method)
		}

		// Verify deployments call
		var deployCall *mockHTTPCall
		for i := range calls {
			if strings.Contains(calls[i].URL, "management.azure.com") {
				deployCall = &calls[i]
				break
			}
		}
		if deployCall == nil {
			t.Fatal("expected a call to management.azure.com")
		}
		if !strings.Contains(deployCall.URL, "test-sub") || !strings.Contains(deployCall.URL, "test-rg") {
			t.Errorf("expected deployment URL to contain subscription and resource group, got: %s", deployCall.URL)
		}
		authHeader := deployCall.Headers.Get("Authorization")
		if authHeader != "Bearer mock-access-token" {
			t.Errorf("expected Authorization header 'Bearer mock-access-token', got: '%s'", authHeader)
		}

		// Verify results
		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 2 {
			t.Fatalf("expected 2 models (succeeded only), got %d: %v", len(azureConfig.Models), azureConfig.Models)
		}

		modelsSet := make(map[string]bool)
		for _, m := range azureConfig.Models {
			modelsSet[m] = true
		}
		if !modelsSet["my-gpt4"] || !modelsSet["staging-gpt-4o"] {
			t.Errorf("expected models 'my-gpt4' and 'staging-gpt-4o', got %v", azureConfig.Models)
		}
		if modelsSet["creating-deployment"] {
			t.Error("should not contain 'creating-deployment' (provisioning state: Creating)")
		}
	})

	t.Run("should use discovery mode when deployments is empty array with management", func(t *testing.T) {
		mock := withMockHTTP(t)

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockTokenResponse)
			},
		)
		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "management.azure.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockDeploymentsResponse)
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{},
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		azureConfig := providers["azure-openai"]
		modelsSet := make(map[string]bool)
		for _, m := range azureConfig.Models {
			modelsSet[m] = true
		}
		if !modelsSet["my-gpt4"] || !modelsSet["staging-gpt-4o"] {
			t.Errorf("expected discovered models, got %v", azureConfig.Models)
		}

		calls := mock.getCalls()
		if len(calls) != 2 {
			t.Errorf("expected 2 HTTP calls (token + deployments), got %d", len(calls))
		}
	})

	t.Run("should handle pagination when fetching deployments", func(t *testing.T) {
		mock := withMockHTTP(t)

		page1Response := map[string]any{
			"value": []map[string]any{
				{
					"name": "deployment-1",
					"properties": map[string]any{
						"model":             map[string]any{"name": "gpt-4", "version": "0613", "format": "OpenAI"},
						"provisioningState": "Succeeded",
					},
				},
			},
			"nextLink": "https://management.azure.com/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.CognitiveServices/accounts/test-resource/deployments?api-version=2024-10-01&$skiptoken=abc",
		}

		page2Response := map[string]any{
			"value": []map[string]any{
				{
					"name": "deployment-2",
					"properties": map[string]any{
						"model":             map[string]any{"name": "gpt-4o", "version": "2024-05-13", "format": "OpenAI"},
						"provisioningState": "Succeeded",
					},
				},
			},
		}

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockTokenResponse)
			},
		)

		deployCallCount := 0
		var deployCallMu sync.Mutex
		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "management.azure.com")
			},
			func(req *http.Request) *http.Response {
				deployCallMu.Lock()
				deployCallCount++
				count := deployCallCount
				deployCallMu.Unlock()

				if count == 1 {
					return jsonResponseWithBody(200, page1Response)
				}
				return jsonResponseWithBody(200, page2Response)
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := mock.getCalls()
		if len(calls) != 3 {
			t.Errorf("expected 3 HTTP calls (token + 2 deployment pages), got %d", len(calls))
		}

		azureConfig := providers["azure-openai"]
		modelsSet := make(map[string]bool)
		for _, m := range azureConfig.Models {
			modelsSet[m] = true
		}
		if !modelsSet["deployment-1"] || !modelsSet["deployment-2"] {
			t.Errorf("expected models from both pages, got %v", azureConfig.Models)
		}
	})

	t.Run("should return fallback config if token fetch fails", func(t *testing.T) {
		mock := withMockHTTP(t)

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return textResponseWithBody(401, "Unauthorized")
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("expected fallback (no error), got: %v", err)
		}

		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 0 {
			t.Errorf("expected empty models on fallback, got %v", azureConfig.Models)
		}

		_ = mock
	})

	t.Run("should return fallback config if deployments fetch fails", func(t *testing.T) {
		mock := withMockHTTP(t)

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockTokenResponse)
			},
		)
		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "management.azure.com")
			},
			func(req *http.Request) *http.Response {
				return textResponseWithBody(403, "Forbidden")
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providers, err := gw.FetchProviders()
		if err != nil {
			t.Fatalf("expected fallback (no error), got: %v", err)
		}

		azureConfig := providers["azure-openai"]
		if len(azureConfig.Models) != 0 {
			t.Errorf("expected empty models on fallback, got %v", azureConfig.Models)
		}

		_ = mock
	})
}

// ---------------------------------------------------------------------------
// Tests: Token Caching
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_TokenCaching(t *testing.T) {
	t.Run("should cache and reuse tokens", func(t *testing.T) {
		mock := withMockHTTP(t)

		mockTokenResp := map[string]any{
			"token_type":   "Bearer",
			"expires_in":   float64(3600),
			"access_token": "mock-token",
		}

		mockDeploymentsResp := map[string]any{
			"value": []map[string]any{
				{
					"name": "test-deployment",
					"properties": map[string]any{
						"model":             map[string]any{"name": "gpt-4", "version": "0613", "format": "OpenAI"},
						"provisioningState": "Succeeded",
					},
				},
			},
		}

		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "login.microsoftonline.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockTokenResp)
			},
		)
		mock.addHandler(
			func(req *http.Request) bool {
				return strings.Contains(req.URL.String(), "management.azure.com")
			},
			func(req *http.Request) *http.Response {
				return jsonResponseWithBody(200, mockDeploymentsResp)
			},
		)

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Management: &gateways.AzureManagementConfig{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientSecret:   "test-secret",
				SubscriptionID: "test-sub",
				ResourceGroup:  "test-rg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// First call
		_, err = gw.FetchProviders()
		if err != nil {
			t.Fatalf("first fetch failed: %v", err)
		}

		// Second call (token should be cached)
		_, err = gw.FetchProviders()
		if err != nil {
			t.Fatalf("second fetch failed: %v", err)
		}

		calls := mock.getCalls()

		tokenCalls := 0
		deploymentCalls := 0
		for _, c := range calls {
			if strings.Contains(c.URL, "login.microsoftonline.com") {
				tokenCalls++
			}
			if strings.Contains(c.URL, "management.azure.com") {
				deploymentCalls++
			}
		}

		if tokenCalls != 1 {
			t.Errorf("expected 1 token call (cached), got %d", tokenCalls)
		}
		if deploymentCalls != 2 {
			t.Errorf("expected 2 deployment calls, got %d", deploymentCalls)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: BuildURL
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_BuildURL(t *testing.T) {
	t.Run("should return empty string (Azure SDK constructs URLs internally)", func(t *testing.T) {
		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{"gpt-4"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		url, err := gw.BuildURL("azure-openai/gpt-4", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "" {
			t.Errorf("expected empty URL, got '%s'", url)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: GetAPIKey
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_GetAPIKey(t *testing.T) {
	t.Run("should return the configured API key", func(t *testing.T) {
		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "my-test-key",
			Deployments:  []string{"gpt-4"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiKey, err := gw.GetAPIKey("gpt-4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if apiKey != "my-test-key" {
			t.Errorf("expected 'my-test-key', got '%s'", apiKey)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ResolveLanguageModel
// ---------------------------------------------------------------------------

func TestAzureOpenAIGateway_ResolveLanguageModel(t *testing.T) {
	t.Run("should create language model with configured values", func(t *testing.T) {
		// TODO: ResolveLanguageModel is not yet implemented in Go.
		// The TS test verifies that a model is returned; in Go the method
		// returns an error because the AI SDK Azure provider is not available.
		t.Skip("not yet implemented: ResolveLanguageModel requires AI SDK Azure provider")

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			APIVersion:   "2024-04-01-preview",
			Deployments:  []string{"gpt-4"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model, err := gw.ResolveLanguageModel(gateways.ResolveLanguageModelArgs{
			ModelID:    "gpt-4",
			ProviderID: "azure-openai",
			APIKey:     "test-key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
	})

	t.Run("should use default API version if not provided", func(t *testing.T) {
		// TODO: Same as above — not yet implemented.
		t.Skip("not yet implemented: ResolveLanguageModel requires AI SDK Azure provider")

		gw, err := gateways.NewAzureOpenAIGateway(gateways.AzureOpenAIGatewayConfig{
			ResourceName: "test-resource",
			APIKey:       "test-key",
			Deployments:  []string{"gpt-4"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model, err := gw.ResolveLanguageModel(gateways.ResolveLanguageModelArgs{
			ModelID:    "gpt-4",
			ProviderID: "azure-openai",
			APIKey:     "test-key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
	})
}

// Ensure fmt is used (avoid unused import).
var _ = fmt.Sprintf
