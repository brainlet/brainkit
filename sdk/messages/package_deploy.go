package messages

import "encoding/json"

// ── Package Deployment ──

type PackageDeployMsg struct {
	Path     string            `json:"path,omitempty"`     // filesystem path to package dir
	Manifest json.RawMessage   `json:"manifest,omitempty"` // inline manifest JSON
	Files    map[string]string `json:"files,omitempty"`    // inline file map: filename → code (no filesystem needed)
}

func (PackageDeployMsg) BusTopic() string { return "package.deploy" }

type PackageDeployResp struct {
	ResultMeta
	Deployed bool     `json:"deployed"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Services []string `json:"services"`
}

type PackageTeardownMsg struct {
	Name string `json:"name"`
}

func (PackageTeardownMsg) BusTopic() string { return "package.teardown" }

type PackageTeardownResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}

type PackageRedeployMsg struct {
	Path string `json:"path"`
}

func (PackageRedeployMsg) BusTopic() string { return "package.redeploy" }

type PackageRedeployResp struct {
	ResultMeta
	Redeployed bool     `json:"redeployed"`
	Services   []string `json:"services"`
}

type PackageListDeployedMsg struct{}

func (PackageListDeployedMsg) BusTopic() string { return "package.list" }

type PackageListDeployedResp struct {
	ResultMeta
	Packages []DeployedPackageInfo `json:"packages"`
}

type DeployedPackageInfo struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Services []string `json:"services"`
	Status   string   `json:"status"` // "active", "degraded"
}

type PackageDeployInfoMsg struct {
	Name string `json:"name"`
}

func (PackageDeployInfoMsg) BusTopic() string { return "package.info" }

type PackageDeployInfoResp struct {
	ResultMeta
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Services []string `json:"services"`
}
