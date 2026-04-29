package sdk

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
