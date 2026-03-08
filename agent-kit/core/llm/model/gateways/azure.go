// Ported from: packages/core/src/llm/model/gateways/azure.ts
package gateways

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Azure-specific types
// ---------------------------------------------------------------------------

// AzureTokenResponse is the response from Azure AD token endpoint.
type AzureTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

// AzureDeployment represents an Azure OpenAI deployment.
type AzureDeployment struct {
	Name       string `json:"name"`
	Properties struct {
		Model struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Format  string `json:"format"`
		} `json:"model"`
		ProvisioningState string `json:"provisioningState"`
	} `json:"properties"`
}

// AzureDeploymentsResponse is the response from Azure deployments API.
type AzureDeploymentsResponse struct {
	Value    []AzureDeployment `json:"value"`
	NextLink string            `json:"nextLink,omitempty"`
}

// CachedAzureToken holds a cached Azure AD token with expiration.
type CachedAzureToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"` // Unix seconds
}

// AzureManagementConfig holds Azure management API credentials.
type AzureManagementConfig struct {
	TenantID       string `json:"tenantId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
	SubscriptionID string `json:"subscriptionId"`
	ResourceGroup  string `json:"resourceGroup"`
}

// AzureOpenAIGatewayConfig holds configuration for the Azure OpenAI gateway.
type AzureOpenAIGatewayConfig struct {
	ResourceName string                 `json:"resourceName"`
	APIKey       string                 `json:"apiKey"`
	APIVersion   string                 `json:"apiVersion,omitempty"`
	Deployments  []string               `json:"deployments,omitempty"`
	Management   *AzureManagementConfig `json:"management,omitempty"`
}

// ---------------------------------------------------------------------------
// AzureOpenAIGateway
// ---------------------------------------------------------------------------

// AzureOpenAIGateway implements MastraModelGateway for Azure OpenAI.
type AzureOpenAIGateway struct {
	config     AzureOpenAIGatewayConfig
	tokenCache sync.Map // map[string]*CachedAzureToken
}

// Compile-time check that AzureOpenAIGateway implements MastraModelGateway.
var _ MastraModelGateway = (*AzureOpenAIGateway)(nil)

// NewAzureOpenAIGateway creates a new AzureOpenAIGateway with the given config.
func NewAzureOpenAIGateway(config AzureOpenAIGatewayConfig) (*AzureOpenAIGateway, error) {
	gw := &AzureOpenAIGateway{
		config: config,
	}
	if err := gw.validateConfig(); err != nil {
		return nil, err
	}
	return gw, nil
}

// ID implements MastraModelGateway.
func (g *AzureOpenAIGateway) ID() string { return "azure-openai" }

// Name implements MastraModelGateway.
func (g *AzureOpenAIGateway) Name() string { return "azure-openai" }

func (g *AzureOpenAIGateway) validateConfig() error {
	if g.config.ResourceName == "" {
		return fmt.Errorf("resourceName is required for Azure OpenAI gateway")
	}
	if g.config.APIKey == "" {
		return fmt.Errorf("apiKey is required for Azure OpenAI gateway")
	}

	hasDeployments := len(g.config.Deployments) > 0
	hasManagement := g.config.Management != nil

	if hasDeployments && hasManagement {
		// Both deployments and management credentials provided.
		// Using static deployments list and ignoring management API.
	}

	if hasManagement {
		if _, err := g.getManagementCredentials(g.config.Management); err != nil {
			return err
		}
	}

	return nil
}

// FetchProviders implements MastraModelGateway.
func (g *AzureOpenAIGateway) FetchProviders() (map[string]ProviderConfig, error) {
	baseConfig := func(models []string) map[string]ProviderConfig {
		return map[string]ProviderConfig{
			"azure-openai": {
				APIKeyEnvVar: []string{},
				APIKeyHeader: "api-key",
				Name:         "Azure OpenAI",
				Models:       models,
				DocURL:       "https://learn.microsoft.com/en-us/azure/ai-services/openai/",
				Gateway:      "azure-openai",
			},
		}
	}

	if len(g.config.Deployments) > 0 {
		return baseConfig(g.config.Deployments), nil
	}

	if g.config.Management == nil {
		return baseConfig([]string{}), nil
	}

	creds, err := g.getManagementCredentials(g.config.Management)
	if err != nil {
		return baseConfig([]string{}), nil
	}

	token, err := g.getAzureADToken(creds.TenantID, creds.ClientID, creds.ClientSecret)
	if err != nil {
		return baseConfig([]string{}), nil
	}

	deployments, err := g.fetchDeployments(token, creds.SubscriptionID, creds.ResourceGroup, g.config.ResourceName)
	if err != nil {
		return baseConfig([]string{}), nil
	}

	models := make([]string, len(deployments))
	for i, d := range deployments {
		models[i] = d.Name
	}

	return baseConfig(models), nil
}

type managementCredentials struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
	ResourceGroup  string
}

func (g *AzureOpenAIGateway) getManagementCredentials(mgmt *AzureManagementConfig) (*managementCredentials, error) {
	var missing []string
	if mgmt.TenantID == "" {
		missing = append(missing, "tenantId")
	}
	if mgmt.ClientID == "" {
		missing = append(missing, "clientId")
	}
	if mgmt.ClientSecret == "" {
		missing = append(missing, "clientSecret")
	}
	if mgmt.SubscriptionID == "" {
		missing = append(missing, "subscriptionId")
	}
	if mgmt.ResourceGroup == "" {
		missing = append(missing, "resourceGroup")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"management credentials incomplete. Missing: %s. Required fields: tenantId, clientId, clientSecret, subscriptionId, resourceGroup",
			strings.Join(missing, ", "),
		)
	}

	return &managementCredentials{
		TenantID:       mgmt.TenantID,
		ClientID:       mgmt.ClientID,
		ClientSecret:   mgmt.ClientSecret,
		SubscriptionID: mgmt.SubscriptionID,
		ResourceGroup:  mgmt.ResourceGroup,
	}, nil
}

func (g *AzureOpenAIGateway) getAzureADToken(tenantID, clientID, clientSecret string) (string, error) {
	cacheKey := fmt.Sprintf("azure-mgmt-token:%s:%s", tenantID, clientID)

	if cached, ok := g.tokenCache.Load(cacheKey); ok {
		ct := cached.(*CachedAzureToken)
		nowSec := time.Now().Unix()
		if ct.ExpiresAt > nowSec+60 {
			return ct.Token, nil
		}
	}

	tokenEndpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "https://management.azure.com/.default")

	resp, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to get Azure AD token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get Azure AD token: %d %s", resp.StatusCode, string(body))
	}

	var tokenResp AzureTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode Azure AD token response: %w", err)
	}

	expiresAt := time.Now().Unix() + int64(tokenResp.ExpiresIn)
	g.tokenCache.Store(cacheKey, &CachedAzureToken{
		Token:     tokenResp.AccessToken,
		ExpiresAt: expiresAt,
	})

	return tokenResp.AccessToken, nil
}

func (g *AzureOpenAIGateway) fetchDeployments(token, subscriptionID, resourceGroup, resourceName string) ([]AzureDeployment, error) {
	apiURL := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CognitiveServices/accounts/%s/deployments?api-version=2024-10-01",
		subscriptionID, resourceGroup, resourceName,
	)

	var allDeployments []AzureDeployment

	for apiURL != "" {
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create deployments request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Azure deployments: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch Azure deployments: %d %s", resp.StatusCode, string(body))
		}

		var data AzureDeploymentsResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode Azure deployments response: %w", err)
		}
		resp.Body.Close()

		allDeployments = append(allDeployments, data.Value...)
		apiURL = data.NextLink
	}

	// Filter to only successful deployments
	successful := make([]AzureDeployment, 0, len(allDeployments))
	for _, d := range allDeployments {
		if d.Properties.ProvisioningState == "Succeeded" {
			successful = append(successful, d)
		}
	}

	return successful, nil
}

// BuildURL implements MastraModelGateway.
// Azure OpenAI uses the AI SDK createAzure helper which constructs its own URLs,
// so this always returns empty.
func (g *AzureOpenAIGateway) BuildURL(_ string, _ map[string]string) (string, error) {
	return "", nil
}

// GetAPIKey implements MastraModelGateway.
func (g *AzureOpenAIGateway) GetAPIKey(_ string) (string, error) {
	return g.config.APIKey, nil
}

// ResolveLanguageModel implements MastraModelGateway.
// TODO: integrate with actual AI SDK Azure provider when available in Go.
func (g *AzureOpenAIGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	apiVersion := g.config.APIVersion
	if apiVersion == "" {
		apiVersion = "2024-04-01-preview"
	}

	// TODO: call createAzure({ resourceName, apiKey, apiVersion, useDeploymentBasedUrls: true })(modelId)
	// when the Go AI SDK Azure provider is available.
	_ = apiVersion
	_ = args.APIKey

	return nil, fmt.Errorf(
		"AzureOpenAIGateway.ResolveLanguageModel not yet implemented in Go; "+
			"model=%s, apiVersion=%s",
		args.ModelID, apiVersion,
	)
}

// Ensure int64 math doesn't overflow.
var _ = int64(math.MaxInt64)
