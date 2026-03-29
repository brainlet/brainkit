package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RegistrySource represents one configured registry.
type RegistrySource struct {
	Name      string
	URL       string
	AuthToken string
}

// RegistryClient fetches plugin information from registries.
type RegistryClient struct {
	sources []RegistrySource
	client  *http.Client
}

// NewRegistryClient creates a client that queries the given registries in order.
func NewRegistryClient(sources []RegistrySource) *RegistryClient {
	return &RegistryClient{
		sources: sources,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Search queries all registries for plugins matching the query or capabilities.
func (c *RegistryClient) Search(query string, capabilities []string) ([]PluginSummary, error) {
	var all []PluginSummary
	for _, src := range c.sources {
		index, err := c.fetchIndex(src)
		if err != nil {
			continue // skip failing registries
		}
		for _, p := range index.Plugins {
			if matchesSearch(p, query, capabilities) {
				all = append(all, p)
			}
		}
	}
	return all, nil
}

// FetchManifest retrieves the full manifest for a specific plugin.
func (c *RegistryClient) FetchManifest(owner, name, version string) (*PluginManifest, error) {
	for _, src := range c.sources {
		url := fmt.Sprintf("%s/plugins/%s/%s/manifest.json", strings.TrimSuffix(src.URL, "/"), owner, name)
		body, err := c.httpGet(url, src.AuthToken)
		if err != nil {
			continue
		}
		var manifest PluginManifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			continue
		}
		if version == "" || manifest.Version == version {
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("plugin %s/%s not found in any registry", owner, name)
}

func (c *RegistryClient) fetchIndex(src RegistrySource) (*RegistryIndex, error) {
	url := strings.TrimSuffix(src.URL, "/") + "/index.json"
	body, err := c.httpGet(url, src.AuthToken)
	if err != nil {
		return nil, err
	}
	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, err
	}
	return &index, nil
}

func (c *RegistryClient) httpGet(url, authToken string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func matchesSearch(p PluginSummary, query string, capabilities []string) bool {
	if query != "" {
		q := strings.ToLower(query)
		if !strings.Contains(strings.ToLower(p.Name), q) &&
			!strings.Contains(strings.ToLower(p.Description), q) &&
			!strings.Contains(strings.ToLower(p.Owner), q) {
			return false
		}
	}
	if len(capabilities) > 0 {
		for _, wantCap := range capabilities {
			found := false
			for _, haveCap := range p.Capabilities {
				if strings.EqualFold(haveCap, wantCap) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
