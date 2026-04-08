package sdk

import "encoding/json"

// ── Package Deployment ──

type PackageDeployMsg struct {
	Path     string            `json:"path,omitempty"`
	Manifest json.RawMessage   `json:"manifest,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
}

func (PackageDeployMsg) BusTopic() string { return "package.deploy" }

type PackageDeployResp struct {
	ResultMeta
	Deployed bool   `json:"deployed"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Source   string `json:"source"`
}

type PackageTeardownMsg struct {
	Name string `json:"name"`
}

func (PackageTeardownMsg) BusTopic() string { return "package.teardown" }

type PackageTeardownResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}

type PackageListDeployedMsg struct{}

func (PackageListDeployedMsg) BusTopic() string { return "package.list" }

type PackageListDeployedResp struct {
	ResultMeta
	Packages []DeployedPackageInfo `json:"packages"`
}

type DeployedPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Status  string `json:"status"`
}

type PackageDeployInfoMsg struct {
	Name string `json:"name"`
}

func (PackageDeployInfoMsg) BusTopic() string { return "package.info" }

type PackageDeployInfoResp struct {
	ResultMeta
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}
