package deploy

// PackageManifest describes a package (the deployable unit).
type PackageManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description,omitempty"`
	Entry       string        `json:"entry,omitempty"`
	Resolver    string        `json:"resolver,omitempty"`
	Requires    *Requirements `json:"requires,omitempty"`
}

// Requirements declares plugin and secret dependencies.
type Requirements struct {
	Plugins []string `json:"plugins,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
}
