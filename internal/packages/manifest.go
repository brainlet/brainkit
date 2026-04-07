package packages

// PluginManifest describes a plugin in the registry.
type PluginManifest struct {
	Name         string                    `json:"name"`
	Owner        string                    `json:"owner"`
	Version      string                    `json:"version"`
	Description  string                    `json:"description"`
	Capabilities []string                  `json:"capabilities,omitempty"`
	SDKVersion   string                    `json:"sdk_version,omitempty"`
	Platforms    map[string]PlatformBinary `json:"platforms"`
	Config       map[string]ConfigField    `json:"config,omitempty"`
	Signature    string                    `json:"signature,omitempty"`
}

// PlatformBinary describes the binary download for a specific OS/arch.
type PlatformBinary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// ConfigField describes a configuration parameter for a plugin.
type ConfigField struct {
	Type        string `json:"type"` // "string", "secret"
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// RegistryIndex is the top-level index returned by a registry.
type RegistryIndex struct {
	Plugins []PluginSummary `json:"plugins"`
}

// PluginSummary is the abbreviated plugin info in a registry index.
type PluginSummary struct {
	Name         string   `json:"name"`
	Owner        string   `json:"owner"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities,omitempty"`
	ManifestURL  string   `json:"manifest_url,omitempty"`
}

// FullName returns "owner/name".
func (m *PluginManifest) FullName() string {
	return m.Owner + "/" + m.Name
}
