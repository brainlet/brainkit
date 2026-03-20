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
