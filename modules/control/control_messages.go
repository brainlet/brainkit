package control

import (
	"encoding/json"

	bkmodule "github.com/brainlet/brainkit/module"
)

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

// KitModulesMsg requests the live mounted module manifest snapshot.
type KitModulesMsg struct{}

func (KitModulesMsg) BusTopic() string { return "kit.modules" }

// KitModulesResp reports every module currently mounted in the Kit.
type KitModulesResp struct {
	Modules []bkmodule.Descriptor `json:"modules"`
}

// KitModuleMountMsg mounts a registered module into the running Kit.
type KitModuleMountMsg struct {
	ID         string          `json:"id"`
	Config     json.RawMessage `json:"config,omitempty"`
	ConfigYAML string          `json:"configYaml,omitempty"`
}

func (KitModuleMountMsg) BusTopic() string { return "kit.module.mount" }

// KitModuleMountResp reports the mounted module manifest.
type KitModuleMountResp struct {
	Module bkmodule.Descriptor `json:"module"`
}

// KitModuleUnmountMsg unmounts a mounted module from the running Kit.
type KitModuleUnmountMsg struct {
	ID string `json:"id"`
}

func (KitModuleUnmountMsg) BusTopic() string { return "kit.module.unmount" }

// KitModuleUnmountResp reports the module manifest that was unmounted.
type KitModuleUnmountResp struct {
	Module bkmodule.Descriptor `json:"module"`
}

// KitModuleDescribeMsg describes a registered or mounted module.
type KitModuleDescribeMsg struct {
	ID string `json:"id"`
}

func (KitModuleDescribeMsg) BusTopic() string { return "kit.module.describe" }

// KitModuleDescribeResp reports a module manifest and whether it is live.
type KitModuleDescribeResp struct {
	Module  bkmodule.Descriptor `json:"module"`
	Mounted bool                `json:"mounted"`
}
