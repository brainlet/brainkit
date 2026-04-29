package control

// KitSetDrainingMsg toggles runtime draining state.
type KitSetDrainingMsg struct {
	Draining bool `json:"draining"`
}

func (KitSetDrainingMsg) BusTopic() string { return "kit.set-draining" }

// KitSetDrainingResp reports the resulting draining state.
type KitSetDrainingResp struct {
	Draining bool `json:"draining"`
}

// ClusterPeersMsg requests the local cluster identity snapshot.
type ClusterPeersMsg struct{}

func (ClusterPeersMsg) BusTopic() string { return "cluster.peers" }

// ClusterPeersResp reports local control-plane cluster identity.
type ClusterPeersResp struct {
	Peers []ClusterPeerInfo `json:"peers"`
}

// ClusterPeerInfo is the wire shape for cluster.peers.
type ClusterPeerInfo struct {
	ClusterID string `json:"clusterId"`
	RuntimeID string `json:"runtimeId"`
	Namespace string `json:"namespace"`
	CallerID  string `json:"callerId"`
	StartedAt string `json:"startedAt"`
}
