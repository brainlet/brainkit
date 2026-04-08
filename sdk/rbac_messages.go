package sdk

import "encoding/json"

// ── RBAC Administration ──

type RBACAssignMsg struct {
	Source string `json:"source"`
	Role   string `json:"role"`
}

func (RBACAssignMsg) BusTopic() string { return "rbac.assign" }

type RBACAssignResp struct {
	ResultMeta
	Assigned bool `json:"assigned"`
}

type RBACRevokeMsg struct {
	Source string `json:"source"`
}

func (RBACRevokeMsg) BusTopic() string { return "rbac.revoke" }

type RBACRevokeResp struct {
	ResultMeta
	Revoked bool `json:"revoked"`
}

type RBACListMsg struct{}

func (RBACListMsg) BusTopic() string { return "rbac.list" }

type RBACListResp struct {
	ResultMeta
	Assignments []RBACAssignmentInfo `json:"assignments"`
}

type RBACAssignmentInfo struct {
	Source     string `json:"source"`
	Role       string `json:"role"`
	AssignedAt string `json:"assignedAt"`
}

type RBACRolesMsg struct{}

func (RBACRolesMsg) BusTopic() string { return "rbac.roles" }

type RBACRolesResp struct {
	ResultMeta
	Roles json.RawMessage `json:"roles"`
}
