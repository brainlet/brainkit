package sdk

import "encoding/json"

// ── Kit lifecycle messages ──

type KitDeployMsg struct {
	Source      string `json:"source"`
	Code        string `json:"code"`
	Role        string `json:"role,omitempty"`
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

// ── Metrics ──

type MetricsGetMsg struct{}

func (MetricsGetMsg) BusTopic() string { return "metrics.get" }

type MetricsGetResp struct {
	ResultMeta
	Metrics json.RawMessage `json:"metrics"`
}

// ── Peer Discovery ──

type PeersListMsg struct{}

func (PeersListMsg) BusTopic() string { return "peers.list" }

type PeersListResp struct {
	ResultMeta
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
	ResultMeta
	Namespace string `json:"namespace"`
	Address   string `json:"address"`
}

// ── Eval ──

type KitEvalMsg struct {
	Code string `json:"code"`
}

func (KitEvalMsg) BusTopic() string { return "kit.eval" }

type KitEvalResp struct {
	ResultMeta
	Result string `json:"result"`
}

// ── EvalTS (raw TS evaluation in current runtime context) ──

type KitEvalTSMsg struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

func (KitEvalTSMsg) BusTopic() string { return "kit.eval-ts" }

type KitEvalTSResp struct {
	ResultMeta
	Result string `json:"result"`
}

// ── Draining ──

type KitSetDrainingMsg struct {
	Draining bool `json:"draining"`
}

func (KitSetDrainingMsg) BusTopic() string { return "kit.set-draining" }

type KitSetDrainingResp struct {
	ResultMeta
	Draining bool `json:"draining"`
}

// ── Cluster identity ──

type ClusterPeersMsg struct{}

func (ClusterPeersMsg) BusTopic() string { return "cluster.peers" }

type ClusterPeersResp struct {
	ResultMeta
	Peers []ClusterPeerInfo `json:"peers"`
}

type ClusterPeerInfo struct {
	ClusterID string `json:"clusterId"`
	RuntimeID string `json:"runtimeId"`
	Namespace string `json:"namespace"`
	CallerID  string `json:"callerId"`
	StartedAt string `json:"startedAt"`
}

// ── EvalModule (ES module evaluation with import support) ──

type KitEvalModuleMsg struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

func (KitEvalModuleMsg) BusTopic() string { return "kit.eval-module" }

type KitEvalModuleResp struct {
	ResultMeta
	Result string `json:"result"`
}

// ── Send (request-reply from Go) ──

type KitSendMsg struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (KitSendMsg) BusTopic() string { return "kit.send" }

type KitSendResp struct {
	ResultMeta
	Payload json.RawMessage `json:"payload"`
}

// ── Health (bus) ──

type KitHealthMsg struct{}

func (KitHealthMsg) BusTopic() string { return "kit.health" }

type KitHealthResp struct {
	ResultMeta
	Health json.RawMessage `json:"health"`
}
