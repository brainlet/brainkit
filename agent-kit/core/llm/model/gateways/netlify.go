// Ported from: packages/core/src/llm/model/gateways/netlify.ts
package gateways

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Netlify-specific types
// ---------------------------------------------------------------------------

// NetlifyProviderResponse describes a single provider in the Netlify gateway response.
type NetlifyProviderResponse struct {
	TokenEnvVar string   `json:"token_env_var"`
	URLEnvVar   string   `json:"url_env_var"`
	Models      []string `json:"models"`
}

// NetlifyResponse is the response from the Netlify AI gateway providers endpoint.
type NetlifyResponse struct {
	Providers map[string]NetlifyProviderResponse `json:"providers"`
}

// NetlifyTokenResponse is the response from the Netlify AI gateway token endpoint.
type NetlifyTokenResponse struct {
	Token     string `json:"token"`
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"` // Unix seconds
}

// CachedNetlifyToken holds a cached Netlify token with URL and expiration.
type CachedNetlifyToken struct {
	Token     string `json:"token"`
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expiresAt"` // Unix seconds
}

// netlifyTokenData holds the token and URL pair.
type netlifyTokenData struct {
	Token string
	URL   string
}

// ---------------------------------------------------------------------------
// NetlifyGateway
// ---------------------------------------------------------------------------

// NetlifyGateway implements MastraModelGateway for the Netlify AI Gateway.
type NetlifyGateway struct {
	tokenCache sync.Map // map[string]*CachedNetlifyToken
}

// Compile-time check that NetlifyGateway implements MastraModelGateway.
var _ MastraModelGateway = (*NetlifyGateway)(nil)

// NewNetlifyGateway creates a new NetlifyGateway.
func NewNetlifyGateway() *NetlifyGateway {
	return &NetlifyGateway{}
}

// ID implements MastraModelGateway.
func (g *NetlifyGateway) ID() string { return "netlify" }

// Name implements MastraModelGateway.
func (g *NetlifyGateway) Name() string { return "Netlify AI Gateway" }

// FetchProviders implements MastraModelGateway.
func (g *NetlifyGateway) FetchProviders() (map[string]ProviderConfig, error) {
	resp, err := http.Get("https://api.netlify.com/api/v1/ai-gateway/providers")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Netlify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch from Netlify: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Netlify response: %w", err)
	}

	var data NetlifyResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse Netlify response: %w", err)
	}

	config := ProviderConfig{
		APIKeyEnvVar: []string{"NETLIFY_TOKEN", "NETLIFY_SITE_ID"},
		APIKeyHeader: "Authorization",
		Name:         "Netlify",
		Gateway:      "netlify",
		Models:       []string{},
		DocURL:       "https://docs.netlify.com/build/ai-gateway/overview/",
	}

	// Convert Netlify format to standard format
	for providerID, provider := range data.Providers {
		for _, model := range provider.Models {
			config.Models = append(config.Models, fmt.Sprintf("%s/%s", providerID, model))
		}
	}

	return map[string]ProviderConfig{
		"netlify": config,
	}, nil
}

// BuildURL implements MastraModelGateway.
func (g *NetlifyGateway) BuildURL(routerID string, envVars map[string]string) (string, error) {
	siteID := getEnvOrMap(envVars, "NETLIFY_SITE_ID")
	netlifyToken := getEnvOrMap(envVars, "NETLIFY_TOKEN")

	if netlifyToken == "" {
		return "", fmt.Errorf("missing NETLIFY_TOKEN environment variable required for model: %s", routerID)
	}

	if siteID == "" {
		return "", fmt.Errorf("missing NETLIFY_SITE_ID environment variable required for model: %s", routerID)
	}

	tokenData, err := g.getOrFetchToken(siteID, netlifyToken)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get Netlify AI Gateway token for model %s: %w",
			routerID, err,
		)
	}

	url := tokenData.URL
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	return url, nil
}

func (g *NetlifyGateway) getOrFetchToken(siteID, netlifyToken string) (*netlifyTokenData, error) {
	cacheKey := fmt.Sprintf("netlify-token:%s:%s", siteID, netlifyToken)

	if cached, ok := g.tokenCache.Load(cacheKey); ok {
		ct := cached.(*CachedNetlifyToken)
		nowSec := time.Now().Unix()
		if ct.ExpiresAt > nowSec+60 {
			return &netlifyTokenData{Token: ct.Token, URL: ct.URL}, nil
		}
	}

	url := fmt.Sprintf("https://api.netlify.com/api/v1/sites/%s/ai-gateway/token", siteID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+netlifyToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get Netlify AI Gateway token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get Netlify AI Gateway token: %d %s", resp.StatusCode, string(body))
	}

	var tokenResp NetlifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode Netlify token response: %w", err)
	}

	g.tokenCache.Store(cacheKey, &CachedNetlifyToken{
		Token:     tokenResp.Token,
		URL:       tokenResp.URL,
		ExpiresAt: tokenResp.ExpiresAt,
	})

	return &netlifyTokenData{Token: tokenResp.Token, URL: tokenResp.URL}, nil
}

// GetAPIKey implements MastraModelGateway.
func (g *NetlifyGateway) GetAPIKey(modelID string) (string, error) {
	siteID := os.Getenv("NETLIFY_SITE_ID")
	netlifyToken := os.Getenv("NETLIFY_TOKEN")

	if netlifyToken == "" {
		return "", fmt.Errorf("missing NETLIFY_TOKEN environment variable required for model: %s", modelID)
	}

	if siteID == "" {
		return "", fmt.Errorf("missing NETLIFY_SITE_ID environment variable required for model: %s", modelID)
	}

	tokenData, err := g.getOrFetchToken(siteID, netlifyToken)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get Netlify AI Gateway token for model %s: %w",
			modelID, err,
		)
	}

	return tokenData.Token, nil
}

// ResolveLanguageModel implements MastraModelGateway.
// TODO: integrate with actual AI SDK providers when available in Go.
func (g *NetlifyGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	// In TypeScript, this dispatches to provider-specific factories
	// (createOpenAI, createAnthropic, etc.) with the Netlify baseURL.
	// In Go, these AI SDK provider packages are not yet available.
	return nil, fmt.Errorf(
		"NetlifyGateway.ResolveLanguageModel not yet implemented in Go; "+
			"provider=%s, model=%s",
		args.ProviderID, args.ModelID,
	)
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// getEnvOrMap looks up a key in the provided map first, then falls back to os.Getenv.
func getEnvOrMap(envVars map[string]string, key string) string {
	if envVars != nil {
		if v, ok := envVars[key]; ok {
			return v
		}
	}
	return os.Getenv(key)
}
