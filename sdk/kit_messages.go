package sdk

import "encoding/json"

// ── Kit lifecycle messages ──

type KitDeployMsg struct {
	Source      string `json:"source"`
	Code        string `json:"code"`
	PackageName string `json:"packageName,omitempty"`
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

type KitDeployFileMsg struct {
	Path string `json:"path"`
}

func (KitDeployFileMsg) BusTopic() string { return "kit.deploy.file" }

// ── Responses ──

type KitDeployResp struct {
	Deployed  bool           `json:"deployed"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}


type KitTeardownResp struct {
	Removed int `json:"removed"`
}


type KitRedeployResp struct {
	Deployed  bool           `json:"deployed"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}


type KitListResp struct {
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

// ── Metrics ──

type MetricsGetMsg struct{}

func (MetricsGetMsg) BusTopic() string { return "metrics.get" }

type MetricsGetResp struct {
	Metrics json.RawMessage `json:"metrics"`
}

// ── Peer Discovery ──

type PeersListMsg struct{}

func (PeersListMsg) BusTopic() string { return "peers.list" }

type PeersListResp struct {
	Peers      []PeerInfo `json:"peers"`
	Namespaces []string   `json:"namespaces,omitempty"`
}

type PeerInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Address   string            `json:"address"`
	Meta      map[string]string `json:"meta,omitempty"`
}

type PeersResolveMsg struct {
	Name string `json:"name"`
}

func (PeersResolveMsg) BusTopic() string { return "peers.resolve" }

type PeersResolveResp struct {
	Namespace string `json:"namespace"`
	Address   string `json:"address"`
}

// ── Eval ──
//
// KitEvalMsg is the single unified eval command. Mode selects the
// evaluation strategy; when empty, it is inferred from Source's file
// extension (".ts" → "ts", else "script").
//
// Mode values:
//   - "script"  — deploy Code as a temp .ts, read globalThis.__module_result
//     (Source optional; default behaviour when only Code is provided)
//   - "ts"      — evaluate TS source directly in the current runtime
//     context via kernel.EvalTS(Source, Code); no deploy
//   - "module"  — evaluate as an ES module via kernel.EvalModule — supports
//     import statements
type KitEvalMsg struct {
	Source string `json:"source,omitempty"`
	Code   string `json:"code"`
	Mode   string `json:"mode,omitempty"`
}

func (KitEvalMsg) BusTopic() string { return "kit.eval" }

type KitEvalResp struct {
	Result string `json:"result"`
}

// ── Draining ──

type KitSetDrainingMsg struct {
	Draining bool `json:"draining"`
}

func (KitSetDrainingMsg) BusTopic() string { return "kit.set-draining" }

type KitSetDrainingResp struct {
	Draining bool `json:"draining"`
}

// ── Cluster identity ──

type ClusterPeersMsg struct{}

func (ClusterPeersMsg) BusTopic() string { return "cluster.peers" }

type ClusterPeersResp struct {
	Peers []ClusterPeerInfo `json:"peers"`
}

type ClusterPeerInfo struct {
	ClusterID string `json:"clusterId"`
	RuntimeID string `json:"runtimeId"`
	Namespace string `json:"namespace"`
	CallerID  string `json:"callerId"`
	StartedAt string `json:"startedAt"`
}

// ── Send (request-reply from Go) ──

type KitSendMsg struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (KitSendMsg) BusTopic() string { return "kit.send" }

type KitSendResp struct {
	Payload json.RawMessage `json:"payload"`
}

// ── Health (bus) ──

type KitHealthMsg struct{}

func (KitHealthMsg) BusTopic() string { return "kit.health" }

type KitHealthResp struct {
	Health json.RawMessage `json:"health"`
}
