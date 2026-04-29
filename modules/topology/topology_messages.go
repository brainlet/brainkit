package topology

// PeersListMsg requests every known peer and namespace.
type PeersListMsg struct{}

func (PeersListMsg) BusTopic() string { return "peers.list" }

// PeersListResp reports known peers plus every unique namespace.
type PeersListResp struct {
	Peers      []PeerInfo `json:"peers"`
	Namespaces []string   `json:"namespaces,omitempty"`
}

// PeerInfo is the bus wire shape for a known cross-kit endpoint.
type PeerInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Address   string            `json:"address"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// PeersResolveMsg resolves a peer name to a namespace.
type PeersResolveMsg struct {
	Name string `json:"name"`
}

func (PeersResolveMsg) BusTopic() string { return "peers.resolve" }

// PeersResolveResp reports the resolved namespace.
type PeersResolveResp struct {
	Namespace string `json:"namespace"`
	Address   string `json:"address"`
}
