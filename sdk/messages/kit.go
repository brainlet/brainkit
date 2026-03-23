package messages

// ── Kit lifecycle messages ──

type KitDeployMsg struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

func (KitDeployMsg) BusTopic() string { return "kit.deploy" }

type KitTeardownMsg struct {
	Source string `json:"source"`
}

func (KitTeardownMsg) BusTopic() string { return "kit.teardown" }

type KitListMsg struct{}

func (KitListMsg) BusTopic() string { return "kit.list" }

type KitRedeployMsg struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

func (KitRedeployMsg) BusTopic() string { return "kit.redeploy" }

// ── Responses ──

type KitDeployResp struct {
	ResultMeta
	Deployed  bool           `json:"deployed"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}


type KitTeardownResp struct {
	ResultMeta
	Removed int `json:"removed"`
}


type KitRedeployResp struct {
	ResultMeta
	Deployed  bool           `json:"deployed"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}


type KitListResp struct {
	ResultMeta
	Deployments []DeploymentInfo `json:"deployments"`
}


// ── Shared types ──

type ResourceInfo struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	CreatedAt int64  `json:"createdAt"`
}

type DeploymentInfo struct {
	Source    string         `json:"source"`
	CreatedAt string         `json:"createdAt"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}
