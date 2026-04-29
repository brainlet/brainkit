package module

import "context"

// ClusterPeerInfo is the core runtime identity exposed to control modules.
type ClusterPeerInfo struct {
	ClusterID string `json:"clusterId"`
	RuntimeID string `json:"runtimeId"`
	Namespace string `json:"namespace"`
	CallerID  string `json:"callerId"`
	StartedAt string `json:"startedAt"`
}

// RuntimeControl is the neutral capability shape used by runtime-control
// modules without making core depend on any concrete control module package.
type RuntimeControl interface {
	SetDraining(bool)
	ClusterPeers(context.Context) ([]ClusterPeerInfo, error)
}
